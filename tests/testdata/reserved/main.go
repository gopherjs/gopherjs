package main

import "fmt"

func main() {
	reservedLoopLabel()
	reservedGotoLabel()
	reservedSwitchCaseLabel()
	reservedSwitchLabelInCase()
	reservedLabelledSwitch()
	reservedLabelledBranch()
	reservedSelectLabel()
	reservedTypeSwitchLabel()
}

func reservedLoopLabel() {
	values := []int{}
class:
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			continue class
		}
		if i > 7 {
			break class
		}
		values = append(values, i)
	}
	fmt.Println(`reservedLoopLabel:`, values)
}

func reservedGotoLabel() {
	values := []int{}
	i := 0
class:
	if i < 5 {
		i++
		values = append(values, i)
		goto class
	}
	fmt.Println(`reservedGotoLabel:`, values)
}

func reservedSwitchCaseLabel() {
	values := []int{}
	const class = 2
	for i := 0; i < 3; i++ {
		switch i {
		case class:
			values = append(values, 42)
		default:
			values = append(values, i)
		}
	}
	fmt.Println(`reservedSwitchCaseLabel:`, values)
}

func reservedSwitchLabelInCase() {
	values := []int{}
	for i := 0; i < 5; i++ {
		switch {
		case i < 3:
			if i == 2 {
				goto class
			}
			values = append(values, 99)
			break
		class:
			values = append(values, 42)
		default:
			values = append(values, i)
		}
	}
	fmt.Println(`reservedSwitchLabelInCase:`, values)
}

func reservedLabelledSwitch() {
	values := []int{}
	for i := 0; i < 5; i++ {
	class:
		switch {
		case i < 3:
			i++
			break class
		default:
			values = append(values, i)
		}
	}
	fmt.Println(`reservedLabelledSwitch:`, values)
}

func reservedLabelledBranch() {
	values := []int{}
	i := 0
class:
	if i < 3 {
		i++
		goto class
	} else {
		values = append(values, i)
	}
	fmt.Println(`reservedLabelledBranch:`, values)
}

func reservedSelectLabel() {
	class := make(chan int, 2)
	class <- 42
	values := []int{}
	for i := 0; i < 3; i++ {
		select {
		case v := <-class:
			values = append(values, v)
		default:
			values = append(values, i)
		}
	}
	fmt.Println(`reservedSelectLabel:`, values)
}

type class interface {
	class_() int
}

type classImpl struct{}

func (c *classImpl) class_() int { return 42 }

func reservedTypeSwitchLabel() {
	classImpl := &classImpl{}
	values := []int{}
	for i, v := range []interface{}{classImpl, 7, "string"} {
		switch t := v.(type) {
		case class:
			values = append(values, t.class_())
		default:
			values = append(values, i)
		}
	}
	fmt.Println(`reservedTypeSwitchLabel:`, values)

}
