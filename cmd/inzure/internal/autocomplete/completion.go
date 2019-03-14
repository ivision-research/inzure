package autocomplete

import "fmt"

type Completion struct {
	Completion string
	ShortHelp  string
}

func (c *Completion) Print() {
	if c.Completion == "" {
		return
	}
	fmt.Println(c.Completion)
	if IsZSH {
		if c.ShortHelp == "" {
			fmt.Println("_")
		} else {
			fmt.Println(c.ShortHelp)
		}
	}
}

type Completions []Completion

func (c Completions) Print() {
	for _, comp := range c {
		comp.Print()
	}
}
