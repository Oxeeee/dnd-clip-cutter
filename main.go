package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

var l loggers

type loggers struct {
	logInfo *log.Logger
	logWarn *log.Logger
	logErr  *log.Logger
}

func (l *loggers) info(v ...interface{}) {
	l.logInfo.Println(v...)
}

func (l *loggers) warn(v ...interface{}) {
	l.logWarn.Println(v...)
}

func (l *loggers) err(v ...interface{}) {
	l.logErr.Println(v...)
}

func init() {
	flags := log.LstdFlags | log.Lshortfile

	fileInfo, _ := os.OpenFile("log_info.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	fileWarn, _ := os.OpenFile("log_warn.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	fileErr, _ := os.OpenFile("log_err.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	logInfo := log.New(fileInfo, "INFO:\t", flags)
	logWarn := log.New(fileWarn, "WARN:\t", flags)
	logErr := log.New(fileErr, "ERROR:\t", flags)

	l = loggers{
		logInfo: logInfo,
		logWarn: logWarn,
		logErr:  logErr,
	}

}

func main() {
	gtk.Init(nil)

	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		l.err(err)
	}
	win.SetTitle("MP4 Cutter")
	win.SetDefaultSize(600, 400)

	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	label, _ := gtk.LabelNew("Drag-and-drop MP4 file")
	label.SetHAlign(gtk.ALIGN_CENTER)

	targetEntry, err := gtk.TargetEntryNew("text/uri-list", gtk.TARGET_OTHER_APP, 0)
	if err != nil {
		l.err(err)
	}

	win.DragDestSet(gtk.DEST_DEFAULT_ALL, []gtk.TargetEntry{*targetEntry}, gdk.ACTION_COPY)

	win.Connect("drag-data-received", func(window *gtk.Window, context *gdk.DragContext, x, y int, selection *gtk.SelectionData, info uint, time uint32) {
		droppedFile := string(selection.GetData())
		filePath := strings.TrimPrefix(droppedFile, "file://")
		filePath = strings.TrimSuffix(filePath, ".mp4")
		filePath = strings.TrimSpace(filePath)

		if filepath.Ext(filePath) != ".mp4" {
			l.warn("file doesn't have mp4 ext.")
			dialog := gtk.MessageDialogNew(window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "Only MP4 files are allowed")
			dialog.Run()
			dialog.Destroy()
			return
		}

		startTime, endTime := askForTimes(window)
		if startTime == "" || endTime == "" {
			return
		}

		outputFileName := generateUniqueFileName()

		err := cutVideo(startTime, endTime, filePath)
		if err != nil {
			l.err("error when cutting video", err)
			dialog := gtk.MessageDialogNew(window, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "Error when cutting video")
			dialog.Run()
			dialog.Destroy()
		} else {
			l.info("Video successfully saved")
			dialog := gtk.MessageDialogNew(window, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK, fmt.Sprintf("Video saved as: %s", outputFileName))
			dialog.Run()
			dialog.Destroy()
		}
	})

	win.Add(label)
	win.ShowAll()
	gtk.Main()
}

func askForTimes(window *gtk.Window) (string, string) {
	startTime := showDialog(window, "Enter the start time (hh:mm:ss):")
	if startTime == "" {
		return "", ""
	}
	endTime := showDialog(window, "Enter the end time (hh:mm:ss):")
	if endTime == "" {
		return "", ""
	}
	return startTime, endTime
}

func showDialog(window *gtk.Window, message string) string {
	dialog := gtk.MessageDialogNew(window, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_OK_CANCEL, message)
	contentArea, err := dialog.GetContentArea()
	if err != nil {
		l.err("Error when receiving the contents of a dialog:", err)
		fmt.Println("Error when receiving the contents of a dialog:", err)
		return ""
	}

	entry, _ := gtk.EntryNew()

	contentArea.PackStart(entry, false, false, 0)

	dialog.ShowAll()

	response := dialog.Run()
	var text string
	if response == gtk.RESPONSE_OK {
		text, _ = entry.GetText()
	}

	dialog.Destroy()

	return strings.TrimSpace(text)
}

func cutVideo(start, end, input string) error {
	base := filepath.Base(input)
	ext := filepath.Ext(base)

	timestamp := time.Now().Format("15-04_02-01-06")
	output := fmt.Sprintf("%s-%s%s", timestamp, base[:len(base)-len(ext)], ext)

	cmd := exec.Command("ffmpeg", "-i", input, "-ss", start, "-to", end, "-c", "copy", output)

	err := cmd.Run()
	if err != nil {
		l.err(err)
		return err
	}
	return nil
}

func generateUniqueFileName() string {
	currentTime := time.Now()
	return fmt.Sprintf("output_%s.mp4", currentTime.Format("15-04_02-01-06"))
}
