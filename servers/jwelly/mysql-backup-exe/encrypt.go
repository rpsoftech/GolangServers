package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"filippo.io/age"
)

// encryptedSuffix is appended to a finished backup's file name when the
// database has encryptTo set.
const encryptedSuffix = ".age"

// parseRecipients turns the configured age public keys into recipients.
func parseRecipients(keys []string) ([]age.Recipient, error) {
	recipients := make([]age.Recipient, 0, len(keys))
	for i, k := range keys {
		r, err := age.ParseX25519Recipient(strings.TrimSpace(k))
		if err != nil {
			return nil, fmt.Errorf("encryptTo[%d]: not a valid age public key (expected age1...): %w", i, err)
		}
		recipients = append(recipients, r)
	}
	return recipients, nil
}

// encryptFile writes an encrypted copy of src to dst. It goes through a .part
// file, so a half-written backup is never mistaken for a finished one.
func encryptFile(src, dst string, recipients []age.Recipient) error {
	return writeAtomic(dst, func(w io.Writer) error {
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()
		enc, err := age.Encrypt(w, recipients...)
		if err != nil {
			return err
		}
		if _, err := io.Copy(enc, in); err != nil {
			return err
		}
		return enc.Close()
	})
}

// decryptFile writes the decrypted contents of src to dst, using the secret
// key(s) in identityFile.
func decryptFile(src, dst, identityFile string) error {
	idf, err := os.Open(identityFile)
	if err != nil {
		return fmt.Errorf("cannot read key file: %w", err)
	}
	defer idf.Close()
	identities, err := age.ParseIdentities(idf)
	if err != nil {
		return fmt.Errorf("%s is not a valid age key file: %w", identityFile, err)
	}
	return writeAtomic(dst, func(w io.Writer) error {
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()
		plain, err := age.Decrypt(in, identities...)
		if err != nil {
			return fmt.Errorf("cannot decrypt %s (wrong key, or the file is damaged): %w", src, err)
		}
		_, err = io.Copy(w, plain)
		return err
	})
}

// writeAtomic fills dst through fill, via dst.part. The final close is checked
// because a failed flush must not look like a finished file. An existing dst is
// never overwritten.
func writeAtomic(dst string, fill func(io.Writer) error) (err error) {
	if _, statErr := os.Stat(dst); statErr == nil {
		return fmt.Errorf("%s already exists", dst)
	}
	part := dst + ".part"
	out, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(part)
		}
	}()
	if err = fill(out); err != nil {
		out.Close()
		return err
	}
	if err = out.Close(); err != nil {
		return err
	}
	return os.Rename(part, dst)
}

// generateKeyFile creates a new key pair in path, which must not exist yet, and
// returns the public key to put in encryptTo.
func generateKeyFile(path string) (string, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("cannot create %s (it must not exist yet): %w", path, err)
	}
	_, werr := fmt.Fprintf(f, "# created: %s\n# public key: %s\n%s\n", time.Now().UTC().Format(time.RFC3339), id.Recipient(), id)
	if err := errors.Join(werr, f.Close()); err != nil {
		os.Remove(path)
		return "", err
	}
	return id.Recipient().String(), nil
}

// runSubcommand handles `keygen` and `decrypt`. It reports whether args named
// one, and the exit code if so.
func runSubcommand(args []string, stdout, stderr io.Writer) (handled bool, code int) {
	if len(args) == 0 {
		return false, 0
	}
	switch args[0] {
	case "keygen":
		return true, keygenCommand(args[1:], stdout, stderr)
	case "decrypt":
		return true, decryptCommand(args[1:], stdout, stderr)
	}
	return false, 0
}

func keygenCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "", "file to write the new key to (required, must not exist)")
	if fs.Parse(args) != nil {
		return 2
	}
	if *out == "" {
		fmt.Fprintln(stderr, "usage: mysql-backup-exe keygen -o <keyfile>")
		return 2
	}
	pub, err := generateKeyFile(*out)
	if err != nil {
		fmt.Fprintln(stderr, "ERROR:", err)
		return 1
	}
	fmt.Fprintf(stdout, "Public key (put it in a database's \"encryptTo\"):\n%s\n\n", pub)
	fmt.Fprintf(stdout, "The secret key is in %s. Keep it OFF the backup machine (password manager, printed copy).\n", *out)
	fmt.Fprintln(stdout, "Without it no encrypted backup can be opened.")
	return 0
}

func decryptCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("decrypt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	identity := fs.String("identity", "", "key file made by keygen (required)")
	out := fs.String("o", "", "output file (default: the input without .age)")
	if fs.Parse(args) != nil {
		return 2
	}
	if fs.NArg() != 1 || *identity == "" {
		fmt.Fprintln(stderr, "usage: mysql-backup-exe decrypt --identity <keyfile> [-o <out>] <file.age>")
		return 2
	}
	src := fs.Arg(0)
	dst := *out
	if dst == "" {
		var ok bool
		if dst, ok = strings.CutSuffix(src, encryptedSuffix); !ok {
			fmt.Fprintf(stderr, "ERROR: %s does not end in %s; pass -o to name the output\n", src, encryptedSuffix)
			return 2
		}
	}
	if err := decryptFile(src, dst, *identity); err != nil {
		fmt.Fprintln(stderr, "ERROR:", err)
		return 1
	}
	fmt.Fprintf(stdout, "Decrypted to %s\n", dst)
	return 0
}
