package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
)

// newApp builds an app that maps *interfaces.RequestError to its status code,
// like the real servers do, so the tests see what a client would see.
func newApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			var reqErr *interfaces.RequestError
			if errors.As(err, &reqErr) {
				return c.Status(reqErr.StatusCode).JSON(reqErr)
			}
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		},
	})
}

func status(t *testing.T, app *fiber.App, req *http.Request) int {
	t.Helper()
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	return res.StatusCode
}

func TestRoleChecker(t *testing.T) {
	checker := &RoleCheckerMiddlewareWithRolesArray{roles: []string{"ADMIN", "USER"}}
	cases := []struct {
		name string
		role any // nil leaves the local unset
		want int
	}{
		{"role missing", nil, 403},
		{"role not a string", 42, 403},
		{"allowed role", "ADMIN", 200},
		{"another allowed role", "USER", 200},
		{"god bypasses the list", ROLE_GOD, 200},
		{"role not in the list", "GUEST", 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newApp()
			app.Use(func(c fiber.Ctx) error {
				if tc.role != nil {
					c.Locals(REQ_LOCAL_KEY_ROLE, tc.role)
				}
				return c.Next()
			}, checker.Validate)
			app.Get("/", func(c fiber.Ctx) error { return c.SendString("ok") })

			if got := status(t, app, httptest.NewRequest(http.MethodGet, "/", nil)); got != tc.want {
				t.Errorf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestTokenDecrypter(t *testing.T) {
	t.Cleanup(func() { TokenDecrypterFunctions = nil })

	// run sends a request and reports what the handler after the middleware
	// found in the request-scoped error local.
	run := func(t *testing.T, header string) (code int, localErr any) {
		t.Helper()
		app := newApp()
		app.Use(TokenDecrypter)
		app.Get("/", func(c fiber.Ctx) error {
			localErr = c.Locals(interfaces.REQ_LOCAL_ERROR_KEY)
			return c.SendString("ok")
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			req.Header.Set(env.RequestTokenHeaderKey, header)
		}
		code = status(t, app, req)
		return code, localErr
	}

	t.Run("no token sets the error local and continues", func(t *testing.T) {
		TokenDecrypterFunctions = func(fiber.Ctx, *string) error {
			t.Fatal("decrypter must not run without a token")
			return nil
		}
		code, localErr := run(t, "")
		if code != 200 {
			t.Errorf("status = %d, want 200 (the route decides what to do with the error)", code)
		}
		if re, ok := localErr.(*interfaces.RequestError); !ok || re.StatusCode != 403 {
			t.Errorf("error local = %#v, want a 403 RequestError", localErr)
		}
	})

	t.Run("valid token leaves no error", func(t *testing.T) {
		var got string
		TokenDecrypterFunctions = func(_ fiber.Ctx, token *string) error {
			got = *token
			return nil
		}
		code, localErr := run(t, "abc")
		if code != 200 || localErr != nil {
			t.Errorf("status = %d, error local = %#v", code, localErr)
		}
		if got != "abc" {
			t.Errorf("decrypter saw token %q, want abc", got)
		}
	})

	t.Run("rejected token is stored for the route", func(t *testing.T) {
		reject := &interfaces.RequestError{StatusCode: 401, Message: "bad token"}
		TokenDecrypterFunctions = func(fiber.Ctx, *string) error { return reject }
		code, localErr := run(t, "abc")
		if code != 200 {
			t.Errorf("status = %d, want 200", code)
		}
		if localErr != any(reject) {
			t.Errorf("error local = %#v, want the decrypter's error", localErr)
		}
	})
}
