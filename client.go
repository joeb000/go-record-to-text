package main

/*
  #include <stdio.h>
  #include <unistd.h>
  #include <termios.h>
  char getch(){
      char ch = 0;
      struct termios old = {0};
      fflush(stdout);
      if( tcgetattr(0, &old) < 0 ) perror("tcsetattr()");
      old.c_lflag &= ~ICANON;
      old.c_lflag &= ~ECHO;
      old.c_cc[VMIN] = 1;
      old.c_cc[VTIME] = 0;
      if( tcsetattr(0, TCSANOW, &old) < 0 ) perror("tcsetattr ICANON");
      if( read(0, &ch,1) < 0 ) perror("read()");
      old.c_lflag |= ICANON;
      old.c_lflag |= ECHO;
      if(tcsetattr(0, TCSADRAIN, &old) < 0) perror("tcsetattr ~ICANON");
      return ch;
  }
*/
import "C"
import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"

	"github.com/asticode/go-astideepspeech"
)

func errCheck(err error) {

	if err != nil {
		panic(err)
	}
}

const MAX_RUN_TIME_SECONDS = 20

// Constants
const (
	beamWidth            = 500
	nCep                 = 26
	nContext             = 9
	lmWeight             = 0.75
	validWordCountWeight = 1.85
)

var configDir = flag.String("configDir", "", "Path to config directory for the DeepSpeech Model files")
var model = flag.String("model", "output_graph.pbmm", "File name of the model (protocol buffer binary file)")
var alphabet = flag.String("alphabet", "alphabet.txt", "File name of the configuration file specifying the alphabet used by the network")
var lm = flag.String("lm", "lm.binary", "File name of the language model binary file")
var trie = flag.String("trie", "trie", "File name of the language model trie file created with native_client/generate_trie")

var runTime = flag.Int("rt", 0, "Length of time in seconds to listen for audio in before processing")

func addConfigPath(fileName *string) {
	*fileName = fmt.Sprintf("%v/%v", *configDir, *fileName)
}

func configureFlags() {
	flag.Parse()

	if *configDir == "" {
		fmt.Println("No config Directory Specified")
		flag.PrintDefaults()
		os.Exit(0)
	}

	addConfigPath(model)
	addConfigPath(alphabet)
	addConfigPath(lm)
	addConfigPath(trie)
}

func main() {

	configureFlags()

	// Initialize DeepSpeech
	m := astideepspeech.New(*model, nCep, nContext, *alphabet, beamWidth)
	defer m.Close()
	if *lm != "" {
		m.EnableDecoderWithLM(*alphabet, *lm, *trie, lmWeight, validWordCountWeight)
	}

	// init PortAudio
	errCheck(portaudio.Initialize())
	defer portaudio.Terminate()

	inputChannels := 1
	outputChannels := 0
	sampleRate := 16000
	framesPerBuffer := make([]int16, 64)

	stream, err := portaudio.OpenDefaultStream(inputChannels, outputChannels, float64(sampleRate), len(framesPerBuffer), framesPerBuffer)
	errCheck(err)
	defer stream.Close()

	//Set up stream
	dsStream := astideepspeech.SetupStream(m, 0, uint(sampleRate))

	// start portaudio input stream
	errCheck(stream.Start())

	// Make a channel to send samples through
	c := make(chan []int16, 100)

	// create wait group
	var wg sync.WaitGroup
	wg.Add(1)

	//Start goroutine to stream audio to deepspeech
	go func() {
		defer wg.Done()

		for i := range c {
			dsStream.FeedAudioContent(i, uint(len(i)))
		}

		fmt.Println("\n\nDONE WITH PROCESSING NOW FINISH...")

		// Call finish stream to get the text output from deepspeech
		output := dsStream.FinishStream()

		fmt.Printf("\n\nText: %s\n", output)

	}()

	// boolean to use to stop the execution of the main input stream loop
	breakLoop := false
	quit := make(chan bool)

	// Start goroutine to listen for an input character and send stop signal
	go func() {
		key := 0
		for key != int('q') {
			key = int(C.getch())
		}
		breakLoop = true
		quit <- true

	}()

	// Start goroutine to ensure program stops after the specified run time
	go func() {
		timeCap := *runTime
		if timeCap == 0 {
			timeCap = MAX_RUN_TIME_SECONDS
		}
		time.Sleep(time.Duration(timeCap) * time.Second)
		breakLoop = true
	}()

	displayTicker := []string{
		"-",
		"\\",
		"/",
		"|",
	}
	i := 0
	// important that we minimize operations inside the loop so we don't overflow audio input buffer
	for {
		i++
		if i == 80 {
			i = 0
		}
		if breakLoop {
			break
		}

		errCheck(stream.Read())

		// send copy of buffered input to chan for processing
		copyFPB := make([]int16, len(framesPerBuffer))
		copy(copyFPB, framesPerBuffer)
		c <- copyFPB

		//fmt.Printf("\rListening... Say something to your microphone!  [%v]  |  len= %v | cap= %v", ct, len(c), cap(c))
		fmt.Printf("\rListening... Say something to your microphone! (press 'q' to quit) [%v]", displayTicker[i%4])

	}

	close(c)

	wg.Wait()

	// just so the user's terminal doesn't blow up because of the getCh()
	fmt.Printf("\nPress 'q' to exit program...")
	<-quit
	fmt.Println("Done")
}
