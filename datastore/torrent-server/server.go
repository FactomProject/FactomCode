package main

import (
	"code.google.com/p/gcfg"
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	socks "github.com/hailiang/gosocks"
	"github.com/jackpal/Taipei-Torrent/torrent"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/signal"
	"path"
	"sort"
	"strings"
	"time"
)

const MAX_OUR_REQUESTS = 2

var (
	seedDir       string   = "/tmp/store/seed"
	torrentDir    string   = "/tmp/torrent"
	mlinkfilePath string   = "/tmp/torrent/mlinks/mlinks.csv"
	trackers      string   = "udp://tracker.publicbt.com:80;udp://tracker.openbittorrent.com:80"
	trackerArray  []string = []string{"udp://tracker.publicbt.com:80", "udp://tracker.openbittorrent.com:80"}
	trackerStr    string   = "tr=udp://tracker.publicbt.com:80&tr=udp://tracker.openbittorrent.com:80"
	isFullScan    bool     = true
	logLevel               = "DEBUG"
	minToRun      int      = 10 //to be removed??
)

var (
	cpuprofile    = flag.String("cpuprofile", "", "If not empty, collects CPU profile samples and writes the profile to the given file before the program exits")
	memprofile    = flag.String("memprofile", "", "If not empty, writes memory heap allocations to the given file before the program exits")
	createTorrent = flag.String("createTorrent", "", "If not empty, creates a torrent file from the given root. Writes to stdout")
	createTracker = flag.String("createTracker", "", "Creates a tracker serving the given torrent file on the given address. Example --createTracker=:8080 to serve on port 8080.")

	port                = flag.Int("port", 7777, "Port to listen on. 0 means pick random port. Note that 6881 is blacklisted by some trackers.")
	fileDir             = flag.String("fileDir", seedDir, "path to directory where files are stored")
	seedRatio           = flag.Float64("seedRatio", math.Inf(0), "Seed until ratio >= this value before quitting.")
	useDeadlockDetector = flag.Bool("useDeadlockDetector", true, "Panic and print stack dumps when the program is stuck.")
	useLPD              = flag.Bool("useLPD", true, "Use Local Peer Discovery")
	useUPnP             = flag.Bool("useUPnP", false, "Use UPnP to open port in firewall.")
	useNATPMP           = flag.Bool("useNATPMP", false, "Use NAT-PMP to open port in firewall.")
	gateway             = flag.String("gateway", "", "IP Address of gateway.")
	useDHT              = flag.Bool("useDHT", true, "Use DHT to get peers.")
	trackerlessMode     = flag.Bool("trackerlessMode", false, "Do not get peers from the tracker. Good for testing DHT mode.")
	proxyAddress        = flag.String("proxyAddress", "", "Address of a SOCKS5 proxy to use.")
)

// array sorting implementation
type byDate []os.FileInfo

func (f byDate) Len() int {
	return len(f)
}
func (f byDate) Less(i, j int) bool {
	return f[i].ModTime().Unix() > f[j].ModTime().Unix()
}
func (f byDate) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func getSortedFiles(dir string) (files []os.FileInfo, err error) {
	t, err := ioutil.ReadDir(dir)
	sort.Sort(byDate(t))
	return t, err
}

// get the input parameters from command line to overwrite the ones from config file
func parseTorrentFlags() *torrent.TorrentFlags {
	return &torrent.TorrentFlags{
		Dial:                dialerFromFlags(),
		Port:                *port,
		FileDir:             *fileDir,
		SeedRatio:           *seedRatio,
		UseDeadlockDetector: *useDeadlockDetector,
		UseLPD:              *useLPD,
		UseDHT:              *useDHT,
		UseUPnP:             *useUPnP,
		UseNATPMP:           *useNATPMP,
		TrackerlessMode:     *trackerlessMode,
		// IP address of gateway
		Gateway: *gateway,
	}
}

// load config from config file
func loadConfigurations() {
	// the data structure should be the same as the config file
	cfg := struct {
		App struct {
			SeedDir       string
			TorrentDir    string
			MlinkfilePath string
			IsFullScan    bool
			MinToRun      int
		}
		Torrent struct {
			Port                int
			SeedRatio           float64
			UseDeadlockDetector bool
			UseLPD              bool
			UseUPnP             bool
			UseNATPMP           bool
			Gateway             string
			UseDHT              bool
			TrackerlessMode     bool
			ProxyAddress        string
			Trackers            string
		}
		Log struct {
			LogLevel string
		}
	}{}

	wd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	err = gcfg.ReadFileInto(&cfg, wd+"/torrent-server.conf")
	if err != nil {
		log.Println(err)
	}

	//setting the variables by the valued form the config file
	seedDir = cfg.App.SeedDir
	torrentDir = cfg.App.TorrentDir
	mlinkfilePath = cfg.App.MlinkfilePath
	isFullScan = cfg.App.IsFullScan
	minToRun = cfg.App.MinToRun

	*port = cfg.Torrent.Port
	*seedRatio = cfg.Torrent.SeedRatio
	*useDeadlockDetector = cfg.Torrent.UseDeadlockDetector
	*useLPD = cfg.Torrent.UseLPD
	*useUPnP = cfg.Torrent.UseUPnP
	*useNATPMP = cfg.Torrent.UseNATPMP
	*gateway = cfg.Torrent.Gateway
	*useDHT = cfg.Torrent.UseDHT
	*trackerlessMode = cfg.Torrent.TrackerlessMode
	*proxyAddress = cfg.Torrent.ProxyAddress

	//build the tracker string for torrent file
	trackerArray = strings.Split(cfg.Torrent.Trackers, ";")
	if len(trackerArray) > 0 {
		trackerStr = "tr=" + trackerArray[0]
		for i := 1; i < len(trackerArray); i++ {
			trackerStr = trackerStr + "&tr=" + trackerArray[i]
		}
	}

	logLevel = cfg.Log.LogLevel

}

// get torrent files from torrentDir
func getTorrentFiles() (fileList *[]string, err error) {

	fiList, err := ioutil.ReadDir(torrentDir)
	if err != nil {
		return nil, err
	}
	var count int = 0
	var total int = len(fiList)
	for i := 0; i < total; i++ {
		if !fiList[i].IsDir() {
			count++
		}
	}
	var torrentFileList []string = make([]string, count)
	var j int = 0
	for i := 0; i < total; i++ {
		if !fiList[i].IsDir() {
			//build the full file path
			torrentFileList[j] = path.Join(torrentDir, fiList[i].Name())
			j++
		}
	}
	return &torrentFileList, nil
}

func init() {

	loadConfigurations()

}
func main() {

	var torrentFileChan = make(chan string)
	torrentFlags := parseTorrentFlags()

	//get the existing torrent files
	torrentFileList, err := getTorrentFiles()
	//run the torrent sessions for the existing torrent files
	go RunNCTorrents(torrentFlags, *torrentFileList, &torrentFileChan)
	if err != nil {
		fmt.Println(err)
	}

	//wait for new notary blocks and create additional torrent files
	go CreateNCTorrentFiles(&torrentFileChan)
	if err != nil {
		fmt.Println("after test")
		fmt.Println(err)
	}

	//to add logic to handle server shutdown ??
	time.Sleep(time.Duration(minToRun) * time.Minute)

	return
}

func CreateNCTorrentFiles(torrentFileChan *chan string) (err error) {

	log.Printf("Seed directory: %s", seedDir)
	var isNotExisting bool = true

	for {
		fiList, err := getSortedFiles(seedDir)
		if err != nil {
			return err
		}

		for _, seedFile := range fiList {
			if !seedFile.IsDir() {
				torrentFile := path.Join(torrentDir, seedFile.Name()+".torrent")
				isNotExisting, err = fileNotExists(torrentFile)
				if isNotExisting {
					var seedFilePath string = path.Join(seedDir, seedFile.Name())
					err = createTorrentFile(torrentFile, seedFilePath, trackerArray[0], trackerStr)
					if err != nil {
						return err
					}
					*torrentFileChan <- torrentFile
					// Will check every file in the directory if full scan
				} else if isFullScan != true {
					break
				}
			}
		}

		// add logic for server shutdown..
		//fmt.Println("sleeping for 30 sec")
		time.Sleep(30 * time.Second)
	}

	return
}

// Create a torrent file and a megnet link per block (seed)
func createTorrentFile(torrentFilePath string, seedFilePath string, announcePath string, trackerstr string) (err error) {
	var metaInfo *torrent.MetaInfo
	metaInfo, err = torrent.CreateMetaInfoFromFileSystem(nil, seedFilePath, 0, false)
	if err != nil {
		return
	}
	metaInfo.Announce = announcePath
	metaInfo.CreatedBy = "Notary Chains"
	var torrentFile *os.File
	torrentFile, err = os.Create(torrentFilePath)
	if err != nil {
		return err
	}
	defer torrentFile.Close()
	err = metaInfo.Bencode(torrentFile)
	if err != nil {
		return err
	}

	//write the mlinks to a csv file:
	mlinkfile, err := os.OpenFile(mlinkfilePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer mlinkfile.Close()
	writer := csv.NewWriter(mlinkfile)

	//csv header: block_id,info_hash,mlink,time_stamp
	_, filename := path.Split(seedFilePath)

	record := []string{filename, fmt.Sprintf("%x", metaInfo.InfoHash),
		fmt.Sprintf("magnet:?xt=urn:btih:%x&%s", metaInfo.InfoHash, trackerstr),
		fmt.Sprintf(time.Now().Format(time.RFC3339))}
	writer.Write(record)
	writer.Flush()

	fmt.Printf("magnet:?xt=urn:btih:%x&%s", metaInfo.InfoHash, trackers)
	return
}

func fileNotExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return true, nil
	}
	return err != nil, err
}

func RunNCTorrents(flags *torrent.TorrentFlags, torrentFiles []string, torrentFileChan *chan string) (err error) {
	conChan, listenPort, err := torrent.ListenForPeerConnections(flags)
	if err != nil {
		log.Println("Couldn't listen for peers connection: ", err)
		return
	}
	quitChan := listenSigInt()

	doneChan := make(chan *torrent.TorrentSession)

	torrentSessions := make(map[string]*torrent.TorrentSession)

	lpd := &torrent.Announcer{}

	if len(torrentFiles) > 0 {
		for _, torrentFile := range torrentFiles {
			var ts *torrent.TorrentSession
			ts, err = torrent.NewTorrentSession(flags, torrentFile, uint16(listenPort))
			if err != nil {
				log.Println("Could not create torrent session.", err)
				return
			}
			log.Printf("Starting torrent session for %x", ts.M.InfoHash)
			torrentSessions[ts.M.InfoHash] = ts
		}

		for _, ts := range torrentSessions {
			go func(ts *torrent.TorrentSession) {
				ts.DoTorrent()
				//doneChan <- ts // need morning testing ??
			}(ts)
		}

		if flags.UseLPD {
			lpd = startLPD(torrentSessions, uint16(listenPort))
		}
	}

mainLoop:
	for {
		select {
		case ts := <-doneChan:
			delete(torrentSessions, ts.M.InfoHash)
			if len(torrentSessions) == 0 {
				break mainLoop
			}
		case <-quitChan:
			for _, ts := range torrentSessions {
				err := ts.Quit()
				if err != nil {
					log.Println("Failed: ", err)
				} else {
					log.Println("Done")
				}
			}
		case c := <-conChan:
			log.Printf("New bt connection for ih %x", c.Infohash)
			if ts, ok := torrentSessions[c.Infohash]; ok {
				ts.AcceptNewPeer(c)
			}
		case announce := <-lpd.Announces:
			hexhash, err := hex.DecodeString(announce.Infohash)
			if err != nil {
				log.Println("Err with hex-decoding:", err)
				break
			}
			if ts, ok := torrentSessions[string(hexhash)]; ok {
				log.Printf("Received LPD announce for ih %s", announce.Infohash)
				ts.HintNewPeer(announce.Peer)
			}
		case newTorrentFile := <-*torrentFileChan:
			var ts *torrent.TorrentSession
			ts, err = torrent.NewTorrentSession(flags, newTorrentFile, uint16(listenPort))
			if err != nil {
				log.Println("Could not create torrent session.", err)
				return
			}
			log.Printf("Starting torrent session for %x", ts.M.InfoHash)
			torrentSessions[ts.M.InfoHash] = ts

			go func(ts *torrent.TorrentSession) {
				ts.DoTorrent()
				//doneChan <- ts // need more testing ??
			}(ts)

			if flags.UseLPD {
				lpd.Announce(ts.M.InfoHash)
			}

		}

	}
	return
}

func listenSigInt() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	return c
}

func startLPD(torrentSessions map[string]*torrent.TorrentSession, listenPort uint16) (lpd *torrent.Announcer) {
	lpd, err := torrent.NewAnnouncer(listenPort)
	if err != nil {
		log.Println("Couldn't listen for Local Peer Discoveries: ", err)
		return
	} else {
		for _, ts := range torrentSessions {
			lpd.Announce(ts.M.InfoHash)
		}
	}
	return
}

func dialerFromFlags() torrent.Dialer {
	if len(*proxyAddress) > 0 {
		return socks.DialSocksProxy(socks.SOCKS5, string(*proxyAddress))
	}
	return nil
}
