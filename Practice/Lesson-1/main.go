package main

import "fmt"

// func main() {
// 	fmt.Println("Hello, World!")
// }

// func main() {
// 	var a, b int
// 	fmt.Scan(&a, &b)
// 	fmt.Println(a + b)
// }

func main() {
	var a int
	fmt.Scan(&a)
	c := a / 30
	d := a % 30 * 2
	fmt.Println("На часах:", c, "часов и ", d, "минут")
}
