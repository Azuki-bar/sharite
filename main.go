package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/samber/lo"
)

var (
	file = flag.String("f", "file", "")
	port = flag.String("p", "1323", "port")
	goal = flag.Uint("g", 0, "specify goal length")
)

var (
	startPattern    = regexp.MustCompile(`<<<<<.?START`)
	endPattern      = regexp.MustCompile(`>>>>>.?END`)
	excludePatterns = []*regexp.Regexp{
		regexp.MustCompile(`\\[A-Za-z]({.+})+`),
		regexp.MustCompile(`%.+`),
	}
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

	e.Logger.Fatal(e.Start(net.JoinHostPort("", *port)))
}

type Handler struct {
	sync.Mutex
	filePath string
}

func NewReader(filePath string) *Handler {
	return &Handler{filePath: filePath}
}

type CounterResult struct {
	Length int  `json:"length"`
	Goal   uint `json:"goal"`
}

func (h *Handler) Counter(c echo.Context) error {
	h.Lock()
	defer h.Unlock()
	fp, err := os.Open(h.filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer fp.Close()
	count, err := CountFromReader(fp)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}
	val := CounterResult{
		Length: count,
		Goal:   *goal,
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
			replaced := applyReplaceRules(excludePatterns, line)
			length += len([]rune(string(replaced)))
		}
	}
	return length, nil
}

func applyReplaceRules(rules []*regexp.Regexp, target []byte) []byte {
	return lo.Reduce(
		rules,
		func(agg []byte, item *regexp.Regexp, _ int) []byte {
			return item.ReplaceAll(agg, nil)
		},
		target,
	)

}
