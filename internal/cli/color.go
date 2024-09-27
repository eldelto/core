package cli

func Color(msg, colorCode string) string {
	return "\033[" + colorCode + msg + "\033[0m"
}

func Red(msg string) string {
	return Color(msg, "31m")
}

func Green(msg string) string {
	return Color(msg, "32m")
}

func Brown(msg string) string {
	return Color(msg, "33m")
}

func Blue(msg string) string {
	return Color(msg, "34m")
}

func Magenta(msg string) string {
	return Color(msg, "35m")
}
