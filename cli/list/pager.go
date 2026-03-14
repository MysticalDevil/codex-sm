package list

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	cliutil "github.com/MysticalDevil/codexsm/cli/util"
)

func WriteWithPager(out io.Writer, text string, pager bool, pageSize int, hasHeader bool) error {
	if !pager || pageSize <= 0 || !cliutil.IsTerminalWriter(out) {
		_, err := io.WriteString(out, text)
		return err
	}

	trimmed := strings.TrimRight(text, "\n")
	if trimmed == "" {
		return nil
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) <= pageSize {
		_, err := io.WriteString(out, text)
		return err
	}

	header := ""
	footer := ""
	bodyStart := 0

	bodyEnd := len(lines)
	if hasHeader && len(lines) > 0 {
		header = lines[0]
		bodyStart = 1
	}

	if bodyEnd > bodyStart && strings.HasPrefix(cliutil.StripANSI(strings.TrimSpace(lines[bodyEnd-1])), "showing ") {
		footer = lines[bodyEnd-1]
		bodyEnd--
	}

	body := lines[bodyStart:bodyEnd]
	if len(body) == 0 {
		_, err := io.WriteString(out, text)
		return err
	}

	pages := (len(body) + pageSize - 1) / pageSize
	page := 0
	in := bufio.NewReader(os.Stdin)

	renderPage := func(page int) error {
		if _, err := fmt.Fprint(out, "\x1b[H\x1b[2J"); err != nil {
			return err
		}

		start := page * pageSize

		end := start + pageSize
		if end > len(body) {
			end = len(body)
		}

		if header != "" {
			if _, err := fmt.Fprintln(out, header); err != nil {
				return err
			}
		}

		for _, line := range body[start:end] {
			if _, err := fmt.Fprintln(out, line); err != nil {
				return err
			}
		}

		if footer != "" {
			if _, err := fmt.Fprintln(out, footer); err != nil {
				return err
			}
		}

		return nil
	}

	for {
		if err := renderPage(page); err != nil {
			return err
		}

		if page >= pages-1 {
			break
		}

		if _, err := fmt.Fprintf(out, "-- Page %d/%d -- [j next, k back, g first, G last, a all, q quit]: ", page+1, pages); err != nil {
			return err
		}

		choice, err := in.ReadString('\n')
		if err != nil {
			return err
		}

		if _, err := fmt.Fprint(out, "\r\033[2K"); err != nil {
			return err
		}

		nextPage, act := ApplyPagerChoice(page, pages, choice)
		page = nextPage

		if act == PagerActionQuit {
			break
		}

		if act == PagerActionAll {
			if _, err := fmt.Fprint(out, "\x1b[H\x1b[2J"); err != nil {
				return err
			}

			if header != "" {
				if _, err := fmt.Fprintln(out, header); err != nil {
					return err
				}
			}

			for p := page; p < pages; p++ {
				start := p * pageSize

				end := start + pageSize
				if end > len(body) {
					end = len(body)
				}

				for _, line := range body[start:end] {
					if _, err := fmt.Fprintln(out, line); err != nil {
						return err
					}
				}
			}

			if footer != "" {
				if _, err := fmt.Fprintln(out, footer); err != nil {
					return err
				}
			}

			break
		}
	}

	return nil
}

type PagerAction int

const (
	PagerActionContinue PagerAction = iota
	PagerActionQuit
	PagerActionAll
)

func ApplyPagerChoice(page, pages int, rawChoice string) (int, PagerAction) {
	if pages <= 0 {
		return page, PagerActionQuit
	}

	last := pages - 1

	clean := strings.TrimSpace(rawChoice)
	if clean == "G" {
		return last, PagerActionContinue
	}

	c := strings.ToLower(clean)
	switch c {
	case "q", "quit":
		return page, PagerActionQuit
	case "a", "all":
		return page, PagerActionAll
	case "g", "first", "home":
		return 0, PagerActionContinue
	case "b", "back", "p", "prev", "k":
		if page > 0 {
			return page - 1, PagerActionContinue
		}

		return 0, PagerActionContinue
	case "", "j", "n", "next", " ":
		if page < last {
			return page + 1, PagerActionContinue
		}

		return last, PagerActionContinue
	default:
		if page < last {
			return page + 1, PagerActionContinue
		}

		return last, PagerActionContinue
	}
}
