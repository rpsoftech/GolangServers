package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

func humanDuration(ms int64) string {
	if ms <= 0 {
		return "0s"
	}
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func humanBytes(b int64) string {
	switch {
	case b <= 0:
		return "0b"
	case b < 1<<10:
		return fmt.Sprintf("%db", b)
	case b < 1<<20:
		return fmt.Sprintf("%.1fkb", float64(b)/(1<<10))
	case b < 1<<30:
		return fmt.Sprintf("%.1fmb", float64(b)/(1<<20))
	}
	return fmt.Sprintf("%.1fgb", float64(b)/(1<<30))
}

func printTable(w io.Writer, infos []common.ProcessInfo) {
	if len(infos) == 0 {
		fmt.Fprintln(w, "no processes")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tNAMESPACE\tPID\tUPTIME\tRESTARTS\tSTATUS\tCPU\tMEM")
	for _, i := range infos {
		ns := i.Namespace
		if ns == "" {
			ns = "default"
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d\t%s\t%d\t%s\t%.1f%%\t%s\n",
			i.ID, i.Name, ns, i.PID, humanDuration(i.UptimeMs), i.Restarts, i.Status, i.CPU, humanBytes(i.MemoryB))
	}
	tw.Flush()
}

func printDescribe(w io.Writer, i common.ProcessInfo) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	row := func(k string, v any) { fmt.Fprintf(tw, "%s\t%v\n", k, v) }
	row("name", i.Name)
	row("id", i.ID)
	row("namespace", i.Namespace)
	row("status", i.Status)
	row("pid", i.PID)
	row("restarts", i.Restarts)
	row("unstable restarts", i.Unstable)
	row("uptime", humanDuration(i.UptimeMs))
	row("cpu", fmt.Sprintf("%.1f%%", i.CPU))
	row("memory", humanBytes(i.MemoryB))
	if i.ExitCode != nil {
		row("last exit code", *i.ExitCode)
	}
	row("script", i.Config.Script)
	row("args", strings.Join(i.Config.Args, " "))
	row("interpreter", i.Config.Interpreter)
	row("exec cwd", i.Config.Cwd)
	row("autorestart", i.Config.AutorestartEnabled())
	row("max restarts", i.Config.MaxRestartsOrDefault())
	row("min uptime", fmt.Sprintf("%dms", i.Config.MinUptimeOrDefault()))
	row("kill timeout", fmt.Sprintf("%dms", i.Config.KillTimeoutOrDefault()))
	row("out log", i.OutLog)
	row("error log", i.ErrLog)
	row("created", time.UnixMilli(i.CreatedAt).Format(time.RFC3339))
	tw.Flush()
}
