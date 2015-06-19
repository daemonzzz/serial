package main
import (
	"bufio"
	"fmt"
	"github.com/ohisama/serial"
	"os"
)
func main() {
	device := "COM5"
	baud := 9600
	fmt.Println("open", device, "at", baud)
	port, err := serial.Open(device, baud)
	if err != nil {
		fmt.Println("open failed:", err)
		return
	}
	defer port.Close()
	fmt.Println("ready")
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 100)
	for scanner.Scan() {
		n, err := port.Write([]byte(scanner.Text() + "\r"))
		if err != nil {
			fmt.Println("serial write error:", err)
		}
		fmt.Println("Sent -> ", n)
		n, err = port.Read(buf)
		if err != nil {
			fmt.Println("serial read error:", err)
			break
		}
		fmt.Println(n, "-> read ", string(buf[:n]))
	}
}
