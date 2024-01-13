package main

import (
	"crypto/subtle"
	"errors"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const idLength = 4
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func hasOnlyAllowedChars(inputStr string) bool {
	for _, char := range inputStr {
		if !strings.ContainsRune(letterBytes, char) {
			return false
		}
	}
	return true
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	_, err := os.Stat("data/")
	if err != nil {
		err := os.Mkdir("data", 0755)
		if err != nil {
			return
		}
	}

	renderer := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	e := echo.New()
	// e.Debug = true
	e.Renderer = renderer

	e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		// Be careful to use constant time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(username), []byte(os.Getenv("POTATO_USERNAME"))) == 1 &&
			subtle.ConstantTimeCompare([]byte(password), []byte(os.Getenv("POTATO_PASSWORD"))) == 1 {
			return true, nil
		}
		return false, nil
	}))

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "form.html", "")
	})

	e.POST("/", func(c echo.Context) error {
		paste := c.FormValue("paste")

		if len(strings.TrimSpace(paste)) != 0 {
			var filename string

			maxTries := 5
			for maxTries > 0 {
				filename = randStringBytes(idLength)
				if _, err := os.Stat("data/" + filename); err == nil {
					// file exists, retry
				} else if errors.Is(err, os.ErrNotExist) {
					// file does not exist, so filename is good
					break
				}

				maxTries -= 1
			}

			err := os.WriteFile("data/"+filename, []byte(paste), 0644)
			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}

			return c.Redirect(http.StatusMovedPermanently, "/"+filename)
		}

		return c.Redirect(http.StatusMovedPermanently, "/")
	})

	e.GET("/:paste", func(c echo.Context) error {
		paste := c.Param("paste")
		if !hasOnlyAllowedChars(paste) {
			return c.String(http.StatusNotFound, "Not found")
		}

		data, err := os.ReadFile("data/" + paste)
		if err != nil {
			return c.String(http.StatusNotFound, "Not found")
		}

		return c.Render(http.StatusOK, "display.html", string(data))
	})

	e.Logger.Fatal(e.Start(":1323"))
}
