/*
* photoctl
* 	-d DEVICE
* 	connect ip:port
* 		[-s file] save connection to file if successful
* 	status
* 		print printer status
* 	version
* 		print version
* 	list-files
* 		list files on usb drive
* 	download FILE
* 	upload FILE (TODO: What happens, if memory is full?)
* 	beep -> M300
* 	start REMOTEFILE
* 	stop
* 	pause
* 	delete REMOTEFILE
* 	shell
* 		start an interactive shell
*	reset
* 		this will reset to factory defaults
* 		for inspection old config us dumped and transfered
* 		[-f] force reset even if config cannot be written
* 	top-fan [on | off | during-print | pwm SPEED]
* 	bottom-fan [on | off | during-print]
* 	folder-support (TODO: still possible to down and upload?)
* 	boot-screen TIME
* 	screen-saver STANDBYTIME (0 = off)
* 	save-config
* 	gcode GCODE
* 		send raw gcode
* 	device-name
* 	temperature
*
* 	Ideas:
* 		logging functionality
* 		print time estimation
* 		estimation
* 		alias functionality
* 		run gcode before and after every print
*
* THE PRINTER IS LIKE A LONELY OLD LADY. SHE WILL TALK WITH ANYONE ABOUT
* EVERYTHING.
*
* Problem: Conn is public (due to serialization by encoding/json)
* 	but it's saved value is not needed
*
*
*
 */

package main

import "fmt"
import "encoding/json"
import "time"
import "regexp"
import "encoding/binary"
import "os"
import "os/user"
import "io/ioutil"
import "net"
import "strings"
import "bufio"
import "flag"
import "io"
import "strconv"

// Printer defines a physical printer
type Printer struct {
	Addr string
	//mac      string
	//firmware string
	//id       string
	Name string
	conn net.Conn
}

func errCheck(err error, message string) {
	if err != nil {
		fmt.Printf("%s Error: %s\n", message, err.Error())
	}
}

// Ping sends a command and expects a return
func (p *Printer) Ping() bool {
	p.SendGcode("M99999")
	if p.Read()[0:2] == "ok" {
		return true
	}
	return false
}

func (p *Printer) Connect(target string) bool {
	duration, err := time.ParseDuration("1000ms")
	errCheck(err, "Couldn't parse duration.")
	conn, err := net.DialTimeout("udp", target, duration)
	if err != nil {
		fmt.Println("Connection failed. Error %s\n", err.Error())
		os.Exit(1)
	}
	_, err = conn.Write([]byte("M99999"))
	errCheck(err, "Couldn't write to device.")
	var buf [512]byte
	_, err = conn.Read(buf[0:])
	// TODO: parse this string properly
	//fmt.Printf("Printer returned: %s", string(buf[0:n]))

	p.conn = conn
	p.Addr = target

	usr, _ := user.Current()
	f, err := os.Create(usr.HomeDir + "/.photos")
	if err != nil {
		fmt.Println("Couldn't create file...")
	}
	defer f.Close()
	ser, err := json.MarshalIndent(p, "", "\t")
	errCheck(err, "Couldn't serialize.")
	_, err = f.Write(ser)
	//err = ioutil.WriteFile("~/.photos", []byte(target), 0644)
	if err != nil {
		fmt.Printf("Couldn't write file. Err %s.\n", err)
	}
	return true
}

func (p Printer) shell() {

	fmt.Println("Starting shell to Anycubic Photon")

	ip := "192.168.2.111"
	port := "3000"

	// ping first
	reader := bufio.NewReader(os.Stdin)

	// no timeout needed here
	conn, err := net.Dial("udp", ip+":"+port)
	if err != nil {
		fmt.Println("Connection failed.")
		os.Exit(1)
	}

	message := "M4002"
	bytes := []byte(message)
	_, err = conn.Write(bytes)
	var buf [512]byte

	fmt.Printf("%%:        %s\n", message)

	n, err := conn.Read(buf[0:])
	if err != nil {
		fmt.Println("Error. Aborting.")
		os.Exit(1)
	}

	fmt.Printf("ANYCUBIC: %s", string(buf[0:n]))

	for {
		fmt.Printf("%%:        ")
		text, _ := reader.ReadString('\n')

		text = strings.Replace(text, "\n", "", -1)

		_, err = conn.Write([]byte(text))
		if err != nil {
			fmt.Println("Error. Aborting.")
			os.Exit(1)
		}

		for {
			n, err := conn.Read(buf[0:])
			if err != nil {
				fmt.Println("Error. Aborting.")
				os.Exit(1)
			}

			fmt.Printf("ANYCUBIC: %s\n", strings.TrimRight(string(buf[0:n]), "\n"))
			okstring := string(buf[0:2])
			if okstring == "ok" || string(buf[0:5]) == "Error" || strings.Trim(text, " ") == "M3000" {
				//				fmt.Printf("%s not equal to %s\n", okstring, "ok")
				break
			}
		}
	}
}

func (p *Printer) SendGcode(gcode string) {
	_, err := p.conn.Write([]byte(gcode))
	errCheck(err, "Couldn't send gcode.")
}

func (p *Printer) Read() (msg string) {
	var buf [512]byte
	n, err := p.conn.Read(buf[0:])
	errCheck(err, "Couldn't read from connection.")
	msg = strings.TrimRight(string(buf[0:n]), "\n\r")
	return msg
}

func (p *Printer) readFilelist() (files []string, sizes []int) {
	p.SendGcode("M20")
	for { // while there are still messages
		msg := p.Read()
		if msg[0:2] == "ok" {
			return files, sizes
		}
		if msg == "Begin file list" || msg == "End file list" {
			continue
		}
		index := strings.LastIndex(msg, " ")
		files = append(files, msg[0:index])
		size, err := strconv.Atoi(msg[index+1:])
		errCheck(err, "Couldn't convert size.")
		sizes = append(sizes, size)

	}
}

// TODO clean this, it's dirty as fuck
func (p *Printer) download(file string, path string) {
	p.SendGcode("M6032 '" + file + "'")
	ret := p.Read()
	if ret[0:2] != "ok" {
		fmt.Printf("not OK: %s\n", ret)
		p.SendGcode("M22")
		p.Read()
		p.SendGcode("M6032 '" + file + "'")
		ret = p.Read()
	}
	length, _ := strconv.Atoi(strings.TrimRight(ret[5:], "\n\r"))
	fmt.Printf("Length: %d\n", length)
	var buf [0x500 + 6]byte
	fo, err := os.Create(file)
	defer fo.Close()
	errCheck(err, "Couldn't create output file.")
	var offset int = 0
	counter := 0
	for {
		counter++
		fmt.Printf("Counter: %3d, Progerss %f\n", counter, float32(offset)/float32(length))
		if offset >= length {
			fmt.Println("End reached")
			fo.Close()
			return
		}
		p.SendGcode("M3000")
		n, err := p.conn.Read(buf[0:])
		errCheck(err, "Couldn't readfrom connection.")
		offset += n - 6
		if buf[n-1] != 0x83 {
			fmt.Println("Error 83")
			os.Exit(1)
		}
		maxval := binary.LittleEndian.Uint32(buf[n-7 : n-3])
		fmt.Printf("maxwal : %x\n", maxval)
		var checksum byte = 0
		for i := 0; i < n-2; i++ {
			checksum ^= buf[i]
		}
		if checksum == buf[n-2] {
			fmt.Println("Checksum success.")
		} else {
			fmt.Println("Erroneous checksum detected success.")
			os.Exit(1)
		}
		_, err = fo.Write(buf[:n-6])
		if err != nil && err != io.EOF {
			panic(err)
		}
	}
}

func (p *Printer) info() string {
	p.SendGcode("M99999")
	return p.Read()
}

func (p *Printer) status() float64 {
	p.SendGcode("M27") // twice, because cmd is somewhat faulty
	//p.SendGcode("M27")
	re := regexp.MustCompile(`[0-9]+`)
	numbers := re.FindAllString(p.Read(), 2)
	if len(numbers) != 2 {
		return NaN // == not printing
	}
	//fmt.Println("%s, %s\n", numbers[0], numbers[1])
	cur_bytes, err := strconv.Atoi(numbers[0])
	errCheck(err, "Couldn't convert string to number.")
	total_bytes, err := strconv.Atoi(numbers[1])
	errCheck(err, "Couldn't convert string to number.")
	return float64(cur_bytes) / float64(total_bytes)
}

func (p *Printer) Delete(file string) bool {
	p.SendGcode("M30 " + file)
	msg := p.Read()
	if strings.Contains(msg, "Delete failed") {
		p.Read() // to pop the additional message
		return false
	} else {
		p.Read() // to pop the additional message
		return true
	}
}

func (p *Printer) Beep() {
	p.SendGcode("M300")
	return
}

// TODO clean this up
func (p *Printer) upload(file string) {
	f, err := os.Open(file)
	errCheck(err, "Couldn't read file")
	var buf [0x500 + 6]byte

	// prepare sending
	p.SendGcode("M28 " + file)
	ret := p.Read()
	if ret[0:2] != "ok" {
		fmt.Println("NOT OK")
		panic(1)
	}

	var offset int64 = 0
	fi, _ := f.Stat()
	size := fi.Size()

	var counter int = 0

	start := time.Now()
	for {
		f.Seek(offset, 0)
		var end int64
		if size-offset < 0x500 {
			end = size - offset
		} else {
			end = int64(len(buf) - 6)
		}
		n, err := f.Read(buf[0:end])
		if err != nil {
			fmt.Printf("Error Reading file.")
		}
		binary.LittleEndian.PutUint32(buf[n:n+4], uint32(offset))
		offset += int64(n)
		var checksum byte = 0
		for i := 0; i < n+4; i++ {
			checksum ^= buf[i]
		}
		buf[n+4] = checksum
		buf[n+5] = 0x83
		p.conn.Write(buf[:end+6])
		msg := p.Read()
		progress := float32((counter)*0x500+int(end)) / float32(size)
		elapsed := time.Now()
		speed := float32(counter*0x500) / float32(elapsed.Sub(start).Seconds())
		fmt.Printf("Progress: %02.1f%% Speed %f [byte/s], offset %d, filesize %d\n", 100.0*progress, speed, offset, size)
		counter++
		fmt.Printf("%s\n", msg)
		if msg[0:2] == "ok" {
			if (offset) >= size {
				p.SendGcode("M4012 I1 T" + strconv.Itoa(int(size)))
				msg = p.Read()
				fmt.Printf("%s\n", msg)
				p.SendGcode("M29")
				msg = p.Read()
				fmt.Printf("%s\n", msg)
				fmt.Printf("Total time elapsed: %.2f [s]\n", time.Now().Sub(start).Seconds())
				return
			}
			continue
		}
		//fmt.Printf("resend: %s\n", msg[0:7])
		if msg[0:6] == "resend" {
			//offset_p, err := strconv.Atoi(msg[8:])
			if err != nil {
				fmt.Println("Error.")
				os.Exit(1)
			}
			fmt.Println("Resetting offset")
			offset -= int64(n)
			counter--
		}
	}
}

func (p *Printer) Print(file string) bool {
	p.SendGcode("M6030 '" + file + "'")
	msg := p.Read()
	if msg[0:2] == "ok" {
		return true
	}
	return false
}

// TODO
func readPrinterConfig(p *Printer, path string) {
}

// TODO
func printFilesFormatted() {}

// target: 0 is the lower motherboard fan, 1 is the upper fan
func (p *Printer) Fan(target int, state int) bool {
	switch target {
	case 0: // lower matherboard fan
		if state >= -2 && state <= 1 {
			p.SendGcode("M8030 T" + strconv.Itoa(state))
			return true
		}
		return false
	case 1: // upper fan
		if state >= -2 && state >= 0 {
			p.SendGcode("M8030 I" + strconv.Itoa(state))
			return true
		}
		return false
	default:
		return false
	}
}

func printSubcommands() {
	fmt.Println("connect IP:PORT")
	fmt.Println("\ttest connection to the printer")
	fmt.Println("shell")
	fmt.Println("\topen an interactive shell")
	fmt.Println("info")
	fmt.Println("[downoad | upload | delete ] FILE")
	fmt.Println("list")
	fmt.Println("info")
	fmt.Println("status")
	fmt.Println("print FILE")
	// pause
	// resume
	// stop
	fmt.Println("beep")
	fmt.Println("top-fan [always_off | always_on | during_printing]")
	fmt.Println("bottom-fan [always_off | always_on | during_printing | during led operation]")
	fmt.Println("alias NAME IP:PORT")
	// save
}

func (p *Printer) readDefaults(s *string) bool {
	usr, _ := user.Current()
	data, err := ioutil.ReadFile(usr.HomeDir + "/.photos")
	errCheck(err, "Couldn't read .photon file")
	json.Unmarshal(data, p)
	*s = p.Addr
	//*s = string(place)
	//fmt.Printf("%s\n", *s)
	return true
	// TODO: implement logic
}

func main() {

	connectCmd := flag.NewFlagSet("connect", flag.ExitOnError)
	photonDevice := connectCmd.String("target", "", "Device to connect to.")

	if len(os.Args) < 2 {
		fmt.Println("No subcommand found...")
		printSubcommands()
		os.Exit(1)
	}

	switch os.Args[1] {
	// top-fan [always_on | always_off | during_printing]
	case "top-fan":
		var p Printer
		connectCmd.Parse(os.Args[2:])
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		var err bool
		switch os.Args[2] {
		case "always_off":
			err = p.Fan(1, -1)
		case "always_on":
			err = p.Fan(1, 0)
		case "during_printing":
			err = p.Fan(1, -2)
		}
		if err {
			fmt.Println("Invalid fan setting.")
			os.Exit(1)
		}
		os.Exit(0)
	// bottom-fan [always_on | always_off | during_printing | during
	// led_operation]
	case "bottom-fan": // motherboard fan
		connectCmd.Parse(os.Args[2:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		var err bool
		p.Connect(*photonDevice)
		switch os.Args[2] {
		case "always_off":
			err = p.Fan(0, 0)
		case "always_on":
			err = p.Fan(0, -1)
		case "during_printing":
			err = p.Fan(0, -2)
		case "during_led_operation":
			err = p.Fan(0, 1)
		}
		if err {
			fmt.Println("Invalid fan setting.")
			os.Exit(1)
		}
		os.Exit(0)
		// Print detailed information about the printer.
	case "info":
		connectCmd.Parse(os.Args[2:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		fmt.Println(p.info())
		// Report the printing status.
	case "status":
		connectCmd.Parse(os.Args[2:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		status := p.status()
		if status == NaN {
			fmt.Println("Not printing.")
		} else {
			fmt.Printf("Printing progress: %2.1f%%\n", 100.0*status)
		}
		os.Exit(0)
		// print FILE
		// start to print FILE
	case "print":
		connectCmd.Parse(os.Args[3:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		p.Print(os.Args[2])
		// make the printer beep
	case "beep":
		connectCmd.Parse(os.Args[2:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		p.Beep()
		// connect IP:PORT
		// Check the connection to the printer at IP and PORT
	case "connect":
		connectCmd.Parse(os.Args[2:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		if p.Connect(*photonDevice) {
			fmt.Println("Connection successful.")
		} else {
			fmt.Println("Could not establish a connection")
		}
		// Open an interactive shell (Great for debug and testing gcodes!)
	case "shell":
		connectCmd.Parse(os.Args[2:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.shell()
		// list files on the sd-card
	case "list":
		connectCmd.Parse(os.Args[2:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		//		readPrinterConfig(&p, "/.photos")
		//	p.ping()
		//files := p.filelist()
		//		printFilesFormatted(files)
		files, sizes := p.readFilelist()
		for i := 0; i < len(files); i++ {
			fmt.Printf("Size %10d, \tFile   %s\n", sizes[i], files[i])
		}
		os.Exit(0)
		// download FILE
		// Download FILE from the sd-card
	case "download":
		file := strings.Trim(os.Args[2], "'")
		connectCmd.Parse(os.Args[3:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		//		readPrinterConfig(&p, "/.photos")
		//	p.ping()
		//files := p.filelist()
		//		printFilesFormatted(files)
		p.download(file, "")

		os.Exit(0)
		// upload [FILE]
		// upload [FILE] to the sd-card
	case "upload":
		file := strings.Trim(os.Args[2], "'")
		connectCmd.Parse(os.Args[3:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		//		readPrinterConfig(&p, "/.photos")
		//	p.ping()
		//files := p.filelist()
		//		printFilesFormatted(files)
		p.upload(file)

		os.Exit(0)
		// delete FILE
		// delete FILE from the sd-card
	case "delete":
		file := strings.Trim(os.Args[2], "'")
		connectCmd.Parse(os.Args[3:])
		var p Printer
		if *photonDevice == "" {
			p.readDefaults(photonDevice)
		}
		p.Connect(*photonDevice)
		if p.Delete(file) {
			fmt.Printf("Successfully deleted file '%s'.\n", file)
		} else {
			fmt.Printf("Deletion failed.\n")
		}

	default:
		fmt.Println("No valid subcommand found...")
		printSubcommands()
		os.Exit(1)
	}
	os.Exit(0)
}