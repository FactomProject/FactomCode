package main

import (
	"log"

	"encoding/hex"
	"github.com/jackpal/Taipei-Torrent/torrent"
	"os"

	"flag"
	"math"

	"code.google.com/p/gcfg"
	"os/signal"

	"encoding/csv"
	socks "github.com/hailiang/gosocks"
	"time"
)

const MAX_OUR_REQUESTS = 2

var (
	seedDir            string = "/tmp/client/store/seed"
	torrentDir         string = "/tmp/client/torrent"
	toBeMergedfilePath string = "/tmp/client/torrent/mlinks/tobemerged.csv"
	incomingfilePath   string = "/tmp/client/torrent/mlinks/incoming.csv"
	mlinkfilePath      string = "/tmp/client/torrent/mlinks/mlink.csv"
	trackers           string = "udp://tracker.publicbt.com:80;udp://tracker.openbittorrent.com:80"
	isFullScan         bool   = true
	logLevel                  = "DEBUG"
	minToRun           int    = 10 //to be removed??
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

func loadConfigurations() {
	cfg := struct {
		App struct {
			SeedDir            string
			TorrentDir         string
			MlinkfilePath      string
			ToBeMergedfilePath string
			IncomingfilePath   string
			MinToRun           int
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
		}
		Log struct {
			LogLevel string
		}
	}{}

	wd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	err = gcfg.ReadFileInto(&cfg, wd+"/torrent-client.conf")
	if err != nil {
		log.Println(err)
	}

	//setting the variables by the valued form the config file
	seedDir = cfg.App.SeedDir
	torrentDir = cfg.App.TorrentDir
	toBeMergedfilePath = cfg.App.ToBeMergedfilePath
	incomingfilePath = cfg.App.IncomingfilePath
	mlinkfilePath = cfg.App.MlinkfilePath
	minToRun = cfg.App.MinToRun // to be removed??

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

	logLevel = cfg.Log.LogLevel
	log.Println("cfg:" + seedDir + torrentDir)
}

func init() {

	loadConfigurations()

	//merge files if toBeMergedFile exists
	isNothingToMerge, err := fileNotExists(toBeMergedfilePath)

	if isNothingToMerge {
		return
	}

	//read the mlinks from the saved csv file:
	toBeMergedfile, err := os.Open(toBeMergedfilePath)
	if err != nil {
		panic(err)
	}
	defer toBeMergedfile.Close()

	newReader := csv.NewReader(toBeMergedfile)

	//csv header: block_id,info_hash,mlink,time_stamp
	newRecords, err := newReader.ReadAll()

	if len(newRecords) <= 0 {
		return
	}

	//read the mlinks from the saved csv file:
	oldLinkFile, err := os.Open(mlinkfilePath)
	if err != nil {
		panic(err)
	}
	defer oldLinkFile.Close()

	oldReader := csv.NewReader(oldLinkFile)

	//csv header: block_id,info_hash,mlink,time_stamp
	oldRecords, err := oldReader.ReadAll()

	var totalOldRec int = len(oldRecords)

	for _, newRecord := range newRecords {
		isFound, index, err := findOldRecord(&oldRecords, &newRecord)
		if err != nil {
			panic(err)
		}
		if isFound { //update the existing record
			oldRecords[index][1] = newRecord[1]
			oldRecords[index][2] = newRecord[2]
			oldRecords[index][3] = newRecord[3]
		} else if index == -1 { //not found
			if totalOldRec == 0 { //empty file
				oldRecords = append(oldRecords, newRecord)
			} else {
				slice := oldRecords[0:totalOldRec]
				oldRecords = append(slice, newRecord)
			}
		} else { // to insert the new record right before index
			if index == 0 { //insert into the first record
				slice := [][]string{newRecord}
				oldRecords = append(slice, oldRecords...)
			} else {
				// need to revisit ??
				slice1 := oldRecords[0:index]
				newSlice1 := make([][]string, len(slice1), len(slice1)+1)
				copy(newSlice1, slice1)
				newSlice1 = append(newSlice1, newRecord)

				slice2 := oldRecords[index:totalOldRec]
				oldRecords = append(newSlice1, slice2...)

			}
		}
		totalOldRec = len(oldRecords)

	}

	// rename the old file
	os.Rename(mlinkfilePath, mlinkfilePath+"."+time.Now().Format(time.RFC3339))

	//write the nerged mlinks to a new csv file:
	newMlinkfile, err := os.OpenFile(mlinkfilePath, os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		panic(err)
	}
	defer newMlinkfile.Close()
	writer := csv.NewWriter(newMlinkfile)

	writer.WriteAll(oldRecords)
	writer.Flush()

	// rename the old tobemerged file
	os.Rename(toBeMergedfilePath, toBeMergedfilePath+"."+time.Now().Format(time.RFC3339))

	//create a new empty csv file:
	toBeMergedfile, err = os.OpenFile(toBeMergedfilePath, os.O_CREATE|os.O_WRONLY, 0600)
	toBeMergedfile.Close()
}

func findOldRecord(oldRecords *[][]string, newRecord *[]string) (isFound bool, index int, err error) {
	var total int = len(*oldRecords)

	//need to replace it w/ a better search alg. ??
	for i := 0; i < total; i++ {
		if (*oldRecords)[i][0] == (*newRecord)[0] {
			return true, i, nil
		} else if (*oldRecords)[i][0] > (*newRecord)[0] {
			return false, i, nil
		}

	}
	return false, -1, nil
}

func fileNotExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return true, nil
	}
	return err != nil, err
}

func main() {

	torrentFlags := parseTorrentFlags()

	//need to define a struct ??
	var mLinkChan = make(chan []string)

	mlinkRecords, err := getLocalTorrentLinks()
	if err != nil {
		panic(err)
	}
	go RunNCTorrents(torrentFlags, mlinkRecords, &mLinkChan)

	go checkForNewLinks(&mLinkChan)

	//to add logic to handle server shutdown ??
	time.Sleep(time.Duration(minToRun) * time.Minute)

	return
}

func checkForNewLinks(mLinkChan *chan []string) (err error) {

	for {
		//read the mlinks from the saved csv file:
		incomingfile, err := os.Open(incomingfilePath)
		if err != nil {
			panic(err)
		}
		defer incomingfile.Close()

		reader := csv.NewReader(incomingfile)

		//csv header: block_id,info_hash,mlink,time_stamp
		records, err := reader.ReadAll()

		if len(records) > 0 {
			//write the mlinks to a csv file:
			toBeMergedfile, err := os.OpenFile(toBeMergedfilePath, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				panic(err)
			}
			defer toBeMergedfile.Close()
			writer := csv.NewWriter(toBeMergedfile)

			for _, record := range records {
				*mLinkChan <- record
				writer.Write(record)
			}

			writer.Flush()

			// rename the old file
			os.Rename(incomingfilePath, incomingfilePath+"."+time.Now().Format(time.RFC3339))

			//write the nerged mlinks to a new csv file:
			newIncomingfile, err := os.OpenFile(incomingfilePath, os.O_CREATE|os.O_WRONLY, 0600)

			if err != nil {
				panic(err)
			}
			defer newIncomingfile.Close()
		}

		// add logic for server shutdown..
		//fmt.Println("sleeping for 30 sec")
		time.Sleep(30 * time.Second)

	}

	return
}

func getLocalTorrentLinks() (mlinkRecords *[][]string, err error) {

	//read the mlinks from the saved csv file:
	mlinkfile, err := os.Open(mlinkfilePath)
	if err != nil {
		panic(err)
	}
	defer mlinkfile.Close()

	reader := csv.NewReader(mlinkfile)

	//csv header: block_id,info_hash,mlink,time_stamp
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	return &records, err

}

func RunNCTorrents(flags *torrent.TorrentFlags, mlinkRecords *[][]string, mLinkChan *chan []string) (err error) {
	conChan, listenPort, err := torrent.ListenForPeerConnections(flags)
	if err != nil {
		log.Println("Couldn't listen for peers connection: ", err)
		return
	}
	quitChan := listenSigInt()

	doneChan := make(chan *torrent.TorrentSession)

	torrentSessions := make(map[string]*torrent.TorrentSession)

	lpd := &torrent.Announcer{}

	if len(*mlinkRecords) > 0 {
		for _, mlinkrecord := range *mlinkRecords {
			if torrentSessions[mlinkrecord[1]] == nil {
				var ts *torrent.TorrentSession
				//need to pass the mlink in here instead
				ts, err = torrent.NewTorrentSession(flags, mlinkrecord[2], uint16(listenPort))
				if err != nil {
					log.Println("Could not create torrent session.", err)
					return
				}
				log.Printf("Starting torrent session for %x", ts.M.InfoHash)
				torrentSessions[ts.M.InfoHash] = ts
			}
		}

		for _, ts := range torrentSessions {
			go func(ts *torrent.TorrentSession) {
				ts.DoTorrent()
				//doneChan <- ts
			}(ts)
		}

		if flags.UseLPD {
			lpd = startLPD(torrentSessions, uint16(listenPort))
		}
	} else {
		log.Printf("No mlinks found in file. Waiting for incoming mlinks...")
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
		case newMLinkRecord := <-*mLinkChan:
			if torrentSessions[newMLinkRecord[1]] == nil {
				var ts *torrent.TorrentSession
				ts, err = torrent.NewTorrentSession(flags, newMLinkRecord[2], uint16(listenPort))
				if err != nil {
					log.Println("Could not create torrent session.", err)
					return
				}
				log.Printf("Starting torrent session for %x", ts.M.InfoHash)
				torrentSessions[ts.M.InfoHash] = ts

				go func(ts *torrent.TorrentSession) {
					ts.DoTorrent()
					//doneChan <- ts
				}(ts)

				if flags.UseLPD {
					lpd.Announce(ts.M.InfoHash)
				}
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
