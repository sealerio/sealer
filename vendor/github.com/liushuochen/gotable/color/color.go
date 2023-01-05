package color

import "fmt"

type Color struct {
	Display    int
	Font       int
	Background int
}

// Combine into a terminal escape sequence
func (c *Color) Combine(message string) string {
	value := ""
	if c.Background == 0 {
		value = fmt.Sprintf("\033[%d;%dm%s\033[0m", c.Display, c.Font, message)
	} else {
		value = fmt.Sprintf("\033[%d;%d;%dm%s\033[0m", c.Display, c.Font, c.Background, message)
	}
	return value
}
