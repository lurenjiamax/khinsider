package download

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/marcus-crane/khinsider/v3/pkg/types"
	"github.com/marcus-crane/khinsider/v3/pkg/util"
	"github.com/pterm/pterm"
)

func GetAlbum(album *types.Album) {
	usrHome, _ := os.UserHomeDir()
	downloadFolder := fmt.Sprintf("%s/Downloads", usrHome) // TODO: Offer a configuration option
	directoryPath := fmt.Sprintf("%s/%s", downloadFolder, normaliseFileName(album.Title))
	// TODO: This should be checked before download since it takes ages to get here
	_, err := os.Stat(directoryPath)
	if !errors.Is(err, fs.ErrNotExist) && err != nil {
		panic(err)
	}
	if os.IsExist(err) {
		err := os.Remove(directoryPath)
		if err != nil {
			pterm.Error.Printfln("A folder already exists at %s. Please remove it to continue.", directoryPath)
			os.Exit(1)
		}
	}
	err = os.Mkdir(directoryPath, 0755)
	if os.IsExist(err) && err != nil {
		pterm.Error.Printfln("A folder already exists at %s. Please remove it to continue.", directoryPath)
		os.Exit(1)
	}
	if os.IsNotExist(err) && err != nil {
		panic(err)
	}
	pterm.Success.Printfln("Successfully created %s", directoryPath)
	prevDisc := int32(0)
	trackPadLen := len(fmt.Sprintf("%d", album.Total.Tracks))
	discPadLen := len(fmt.Sprintf("%d", album.Tracks[len(album.Tracks)-1].DiscNumber))
	err = SaveImages(album, directoryPath)
	if err != nil {
		pterm.Error.Printfln("Downloading covers %s", album.Images)
	} else {
		pterm.Success.Printfln("COVERS")
	}
	for _, track := range album.Tracks {
		currDisc := track.DiscNumber
		if currDisc != 0 && currDisc > prevDisc {
			highestTrackNumForDisc := int32(0)
			for _, track := range album.Tracks {
				if track.DiscNumber == currDisc {
					highestTrackNumForDisc = track.TrackNumber
				}
			}
			trackPadLen = len(fmt.Sprintf("%d", highestTrackNumForDisc))
		}
		trackFmt := track.Title
		trackFmt = fmt.Sprintf("%0*d %s", trackPadLen, track.TrackNumber, trackFmt)
		if track.DiscNumber != 0 {
			// TODO: Padding for 10+ discs
			trackFmt = fmt.Sprintf("%0*dx%s", discPadLen, track.DiscNumber, trackFmt)
		}
		err := SaveAudioFile(track, trackFmt, directoryPath)
		if err != nil {
			pterm.Error.Printfln(trackFmt)
		} else {
			pterm.Success.Printfln(trackFmt)
		}
	}
	fmt.Println()

}
func SaveImages(album *types.Album, saveLocation string) error {
	for _, imageURL := range album.Images {
		fileName := filepath.Base(imageURL)
		imageFilePath := filepath.Join(saveLocation, fileName)
		imageURL = strings.Replace(imageURL, "https://delta.vgmsite.com/", "https://vgmsite.com/", 1)
		res, err := util.RequestFile(imageURL)
		if res.StatusCode != http.StatusOK {
			pterm.Debug.Printfln("There was an error downloading %s. received a non-200 status code: %d", imageURL, res.StatusCode)
			return nil
		}
		if err != nil {
			pterm.Debug.Printfln("There was an error downloading %s: %v", imageURL, err)
			return err
		}
		defer func() {
			if res.Body != nil {
				if err := res.Body.Close(); err != nil {
					panic(err)
				}
			}
		}()
		writer, err := os.Create(imageFilePath)
		if err != nil {
			pterm.Debug.Printfln("There was an error creating %s", imageFilePath)
			pterm.Debug.Printfln(err.Error())
			return err
		}
		defer func() {
			if err := writer.Close(); err != nil {
				panic(err)
			}
		}()

		_, err = io.Copy(writer, res.Body)
		if err != nil {
			pterm.Debug.Printfln("There was an error writing %s", fileName)
			return err
		}
		pterm.Debug.Printfln("Successfully downloaded cover %s to %s", imageURL, imageFilePath)
		return nil
	}
	return nil
}

func SaveAudioFile(track types.Track, fileName string, saveLocation string) error {
	trackFile := fmt.Sprintf("%s/%s.flac", saveLocation, normaliseFileName(fileName))
	track.SourceFlac = strings.Replace(track.SourceFlac, "https://delta.vgmsite.com/", "https://vgmsite.com/", 1)
	pterm.Debug.Printfln("Downloading %s", track.SourceFlac)

	res, err := util.RequestFile(track.SourceFlac)
	if res.StatusCode != http.StatusOK {
		pterm.Debug.Printfln("received a non-200 status code: %d", res.StatusCode)
		return nil
	}
	if err != nil {
		pterm.Debug.Printfln("There was an error downloading %s code %v", track.SourceFlac, res.StatusCode)
		return err
	}
	defer func() {
		if res.Body != nil {
			if err := res.Body.Close(); err != nil {
				panic(err)
			}
		}
	}()
	writer, err := os.Create(trackFile)
	if err != nil {
		pterm.Debug.Printfln("There was an error creating %s", trackFile)
		pterm.Debug.Printfln(err.Error())
		return err
	}
	defer func() {
		if err := writer.Close(); err != nil {
			panic(err)
		}
	}()

	_, err = io.Copy(writer, res.Body)
	if err != nil {
		pterm.Debug.Printfln("There was an error writing %s", fileName)
		return err
	}
	return nil
}

func normaliseFileName(title string) string {
	// TODO: Code dump from v1. Should be reviewed again.
	if !utf8.ValidString(title) {
		pterm.Debug.Printfln("Invalid title: %s", title)
		validString := make([]rune, 0, len(title))
		for i, r := range title {
			if r == utf8.RuneError {
				_, size := utf8.DecodeRuneInString(title[i:])
				if size == 1 {
					continue
				}
			}
			validString = append(validString, r)
		}
		pterm.Debug.Printfln("Normalised title: %s", string(validString))
		return string(validString)
	}
	return title
}
