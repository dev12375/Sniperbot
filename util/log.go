package util

const (
	prefix   = "●"
	reset    = "\033[0m"
	red      = "\033[31m"
	green    = "\033[32m"
	yellow   = "\033[33m"
	blue     = "\033[34m"
	magenta  = "\033[35m"
	cyan     = "\033[36m"
	template = "%s%s%s %s\n"
)

// func LogRed(str any) {
// 	fmt.Printf(template, red, prefix, reset, str)
// }
// func LogGreen(str any) {
// 	fmt.Printf(template, green, prefix, reset, str)
// }
// func LogYellow(str any) {
// 	fmt.Printf(template, yellow, prefix, reset, str)
// }

// func LogDebug(str any) {
// 	fmt.Printf(template, blue, prefix, reset, str)
// }
