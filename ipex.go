package main

import (
	"bufio"
	"fmt"
	"github.com/antchfx/xmlquery"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	var (
		imlFile       string
		importDir     string
		filesCopied   = 0
		errorCount    = 0
		currPlaylist  int
		userOs        = runtime.GOOS
		isXmlFound    bool
	)

	// Only run on macOS or Windows
	if userOs == "darwin" || userOs == "windows" {

		// Initialize new reader globally
		in := bufio.NewReader(os.Stdin)

		// Setting up the default path to XML file depending on username and OS
		systemUser, _ := user.Current()
		switch userOs {
		case "darwin": imlFile = "/Users/" + systemUser.Username + "/Music/iTunes/iTunes Music Library.xml"
		case "windows": imlFile = "C:\\Users\\" + strings.Split(systemUser.Username, "\\")[1] + "\\Music\\iTunes\\iTunes Music Library.xml"
		}

	GetXml:

		// Searching for XML
		fmt.Println("Searching for iTunes Media Library XML in:", imlFile, "...", func () string {
			if FileExists(imlFile) {
				isXmlFound = true

				return "\033[32mfound.\033[0m"
			} else {
				isXmlFound = false

				return "\033[31mnot found.\033[0m"
			}
		}())

		// If iTunes Media Library XML file found -- read and parse it
		if isXmlFound {
			playlists := make(map[int]string)

			// Read XML file
			file, err := ioutil.ReadFile(imlFile)

			if err != nil {
				log.Fatal(err)
			}

			str := string(file)

		CreateImportDir:

			// Get the name of import directory from command line
			fmt.Println("Enter the name of import directory:")
			inputId, _ := in.ReadString('\n')
			importDir = strings.TrimRight(inputId, "\r\n")

			// Create import directory
			if !FileExists(importDir) {
				if err := os.Mkdir(importDir, 0777); err != nil {
					fmt.Println("\033[31mCannot create directory.\033[0m")

					goto CreateImportDir
				}
			}

			// Parse XML data...
			var reader= strings.NewReader(str)

			doc, err := xmlquery.Parse(reader)

			if err != nil {
				log.Fatal(err)
			}

			// ... extract playlists
			nodePlaylists := xmlquery.Find(doc, ".//dict/key[.='Playlists']/following-sibling::array//key[.='Name']/following-sibling::string/text()")
			if len(nodePlaylists) > 0 {
				for i, n := range nodePlaylists {
					p := n.InnerText()
					fmt.Println(i+1, "\t", p)
					playlists[i] = p
				}
			} else {
				log.Fatal("\033[33mNo playlists found.\033[0m")
			}

		GetPlaylist:

			// Get a playlist number from command line
			fmt.Println("Enter a playlist number:")
			inputCurrPlaylist, _ := in.ReadString('\n')
			currPlaylist, _ = strconv.Atoi(strings.TrimRight(inputCurrPlaylist, "\r\n"))

			if _, ok := playlists[currPlaylist- 1]; ok {
				fmt.Println("Playlist selected:", playlists[currPlaylist-1])
			} else {
				fmt.Println("\033[31mWrong playlist number.\033[0m")

				goto GetPlaylist
			}

			// ... extract tracks
			nodeTracks := xmlquery.Find(doc, ".//dict/key[.='Playlists']/following-sibling::array//string[.='"+playlists[currPlaylist-1]+"']/following-sibling::array/dict/integer/text()")

			if len(nodeTracks) > 0 {
				fmt.Println("Extracting playlists...")

				for _, n := range nodeTracks {
					p := n.InnerText()

					// Get track location
					trackPath, _ := url.PathUnescape(xmlquery.FindOne(doc, ".//dict/key[.='Tracks']/following-sibling::dict/key[.='"+p+"']/following-sibling::dict/key[.='Location']/following-sibling::string/text()").InnerText())

					b := strings.Split(trackPath, "/")
					dir := importDir

					// Create a directory inside the import one for each file
					for i := 3; i >= 2; i-- {
						dir += "/" + b[len(b)-i]

						if !FileExists(dir) {
							if err := os.Mkdir(dir, 0777); err != nil {
								log.Println(err)
							}
						}
					}

					// Prepare `trackPath`
					t := strings.Replace(trackPath, "file:///", "/", -1)

					// Copy files to the import directory
					if CopyFile(t, importDir+"/"+b[len(b)-3]+"/"+b[len(b)-2]+"/"+b[len(b)-1]) {
						fmt.Println("\033[32mCopied successfully:\033[0m", t)
						filesCopied++
					} else {
						errorCount++
					}
				}

				// Wait a second...
				time.Sleep(time.Second)

				// ... then exit normally
				defer fmt.Println("Done.", len(nodeTracks), "tracks in '"+playlists[currPlaylist-1]+"' playlist.", filesCopied, "files copied.", errorCount, "errors.")
			} else {
				fmt.Println("\033[33mPlaylist is empty.\033[0m")

				goto GetPlaylist
			}
		} else {
			fmt.Println("Enter the path to iTunes Media Library XML file:")
			inputImlFile, _ := in.ReadString('\n')
			imlFile = strings.TrimRight(inputImlFile, "\r\n")

			goto GetXml
		}
	} else {
		log.Fatal("\033[31mUnsupported operating system.\033[0m")
	}
}

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}

func CopyFile(src string, dest string) bool {
	from, err := os.Open(src)

	if err != nil {
		log.Println(err)

		return false
	}

	defer func() {
		err := from.Close()

		if err != nil {
			log.Println(err)
		}
	}()

	to, err := os.OpenFile(dest, os.O_RDWR | os.O_CREATE, 0777)

	if err != nil {
		log.Println(err)

		return false
	}

	defer func() {
		err := to.Close()

		if err != nil {
			log.Println(err)
		}
	}()

	_, err = io.Copy(to, from)

	if err != nil {
		log.Println(err)

		return false
	}

	return true
}