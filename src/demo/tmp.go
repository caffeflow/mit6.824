package main

type Info struct {
	token   string
	numIdle int
}

func main() {
	info := Info{}
	print(len(info.token), info.token == "")
}
