package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	file = flag.String("f", "file", "")
)

var (
	startPattern   = regexp.MustCompile(`% <<<<<START`)
	endPattern     = regexp.MustCompile(`% >>>>>END`)
	excludePattern = regexp.MustCompile(`\\[A-Za-z]({.+})+`)
)

func main() {
	flag.Parse()
	e := echo.New()
	e.Use(middleware.Logger())
	if file == nil {
		log.Print("add file target with -f")
		return
	}
	r := NewReader(*file)
	e.File("/", "public/index.html")
	e.GET("/count", r.Counter)

	e.Logger.Fatal(e.Start(":1323"))
}

type Handler struct {
	mu       *sync.Mutex
	filePath string
}

func NewReader(filePath string) *Handler {
	return &Handler{filePath: filePath, mu: &sync.Mutex{}}
}

func (h *Handler) Counter(c echo.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	fp, err := os.Open(h.filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer fp.Close()
	count, err := CountFromReader(fp)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	val := struct {
		Length int `json:"length"`
	}{
		Length: count,
	}
	return c.JSON(http.StatusOK, val)
}

func CountFromReader(r io.Reader) (int, error) {
	reader := bufio.NewReaderSize(r, 1<<20)
	length := 0
	inArea := false
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if startPattern.Find(line) != nil {
			inArea = true
			continue
		}
		if endPattern.Find(line) != nil {
			inArea = false
			break
		}
		if inArea {
			length += len([]rune(string(excludePattern.ReplaceAll(line, nil))))
		}
	}
	return length, nil
}
