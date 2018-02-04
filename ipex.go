package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"strings"
	"github.com/antchfx/xmlquery"
	"net/url"
	"log"
	"time"
	"runtime"
	"os/user"
	"io"
	"bufio"
	"strconv"
)

func main() {

	var (
		iml_file string
		import_dir string
		files_copied int = 0
		error_count int = 0
		curr_playlist int
		user_os string = runtime.GOOS
		xml_found bool
	)

	// Only run on macOS or Windows
	if user_os == "darwin" || user_os == "windows" {

		// Initialize new reader globally
		in := bufio.NewReader(os.Stdin)

		// Setting up the default path to XML file depending on username and OS
		user, _ := user.Current()
		switch user_os {
		case "darwin": iml_file = "/Users/" + user.Username + "/Music/iTunes/iTunes Music Library.xml"
		case "windows": iml_file = "C:\\Users\\" + strings.Split(user.Username, "\\")[1] + "\\Music\\iTunes\\iTunes Music Library.xml"
		}

	GET_XML:

		// Searching for XML
		fmt.Println("Searching for iTunes Media Library XML in:", iml_file, "...", func () string {
			if FileExists(iml_file) {
				xml_found = true
				return "found."
			} else {
				xml_found = false
				return "not found."
			}
		}())

		// If iTunes Media Library XML file found -- read and parse it
		if xml_found {

			playlists := make(map[int]string)

			// Read XML file
			file, err := ioutil.ReadFile(iml_file)
			if err != nil {
				log.Fatal(err)
			}
			str := string(file)

		CREATE_IMPORT_DIR:

			// Get the name of import directory from command line
			fmt.Println("Enter the name of import directory:")
			input_id, _ := in.ReadString('\n')
			import_dir = strings.TrimRight(input_id, "\r\n")

			// Create import directory
			if !FileExists(import_dir) {
				if err := os.Mkdir(import_dir, 0777); err != nil {
					fmt.Println("Cannot create directory.")
					goto CREATE_IMPORT_DIR
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
				log.Fatal("No playlists found.")
			}

		GET_PLAYLIST:

			// Get a playlist number from command line
			fmt.Println("Enter a playlist number:")
			inp_cp, _ := in.ReadString('\n')
			curr_playlist, _ = strconv.Atoi(strings.TrimRight(inp_cp, "\r\n"))
			if _, ok := playlists[curr_playlist - 1]; ok {
				fmt.Println("Playlist selected:", playlists[curr_playlist-1])
			} else {
				fmt.Println("Wrong playlist number.")
				goto GET_PLAYLIST
			}

			// ... extract tracks
			nodeTracks := xmlquery.Find(doc, ".//dict/key[.='Playlists']/following-sibling::array//string[.='"+playlists[curr_playlist-1]+"']/following-sibling::array/dict/integer/text()")
			if len(nodeTracks) > 0 {
				fmt.Println("Extracting playlists...")
				for _, n := range nodeTracks {
					p := n.InnerText()

					// Get track location
					trackPath, _ := url.PathUnescape(xmlquery.FindOne(doc, ".//dict/key[.='Tracks']/following-sibling::dict/key[.='"+p+"']/following-sibling::dict/key[.='Location']/following-sibling::string/text()").InnerText())

					b := strings.Split(trackPath, "/")
					dir := import_dir

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
					if CopyFile(t, import_dir+"/"+b[len(b)-3]+"/"+b[len(b)-2]+"/"+b[len(b)-1]) {
						fmt.Println("Copied successfully:", t)
						files_copied++
					} else {
						error_count++
					}
				}

				// Wait a second...
				time.Sleep(time.Second)

				// ... then exit normally
				defer fmt.Println("Done.", len(nodeTracks), "tracks in '"+playlists[curr_playlist-1]+"' playlist.", files_copied, "files copied.", error_count, "errors.")
			} else {
				fmt.Println("Playlist is empty.")
				goto GET_PLAYLIST
			}

		} else {
			fmt.Println("Enter the path to iTunes Media Library XML file:")
			input_xml, _ := in.ReadString('\n')
			iml_file = strings.TrimRight(input_xml, "\r\n")
			goto GET_XML
		}

	} else {
		log.Fatal("Unsupported operating system.")
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
	defer from.Close()

	to, err := os.OpenFile(dest, os.O_RDWR | os.O_CREATE, 0777)
	if err != nil {
		log.Println(err)
		return false
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}