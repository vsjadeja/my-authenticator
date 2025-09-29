package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/pbkdf2"
)

const encryptionPassword = "Divorcee0-Equator6-Footman5-Showing7-Scorch3"
const salt = "Dicing1-Stray2-Bulldog2-Prevalent7-Drearily3-Taekwondo8-Diligence2-Balcony3"
const dbFileName = "securedata.bin"

type TOTPEntry struct {
	Title string `json:"title"`
	Hash  string `json:"hash"`
}

// Custom progress widget that can update itself
type AutoProgressBar struct {
	widget.ProgressBar
	animation   *fyne.Animation
	labelWidget *widget.Label
	entries     []TOTPEntry
	codeWidgets []*widget.RichText
}

func NewAutoProgressBar() *AutoProgressBar {
	p := &AutoProgressBar{}
	p.ExtendBaseWidget(p)
	return p
}

func (p *AutoProgressBar) SetLabelWidget(label *widget.Label) {
	p.labelWidget = label
}

func (p *AutoProgressBar) SetTOTPData(entries []TOTPEntry, codeWidgets []*widget.RichText) {
	p.entries = entries
	p.codeWidgets = codeWidgets
}

func (p *AutoProgressBar) StartAutoUpdate() {
	if p.animation != nil {
		p.animation.Stop()
	}

	// Create an animation that runs continuously
	p.animation = fyne.NewAnimation(time.Second*30, func(f float32) {
		now := time.Now()
		secondsIntoInterval := now.Second() % 30
		remaining := 30 - secondsIntoInterval
		progress := float64(secondsIntoInterval) / 30.0
		p.SetValue(1.0 - progress) // Countdown from 1 to 0

		// Update the label text if it exists
		if p.labelWidget != nil {
			p.labelWidget.SetText(fmt.Sprintf("Next refresh in %d seconds", remaining))
		}

		// Update TOTP codes when countdown reaches zero (new cycle starts)
		if remaining == 30 {
			p.updateTOTPCodes()
		}
	})

	// Make the animation repeat indefinitely
	p.animation.RepeatCount = fyne.AnimationRepeatForever
	p.animation.Start()
}

func (p *AutoProgressBar) updateTOTPCodes() {
	if p.entries == nil || p.codeWidgets == nil {
		return
	}

	for i, entry := range p.entries {
		if i < len(p.codeWidgets) {
			code, err := generateTOTP(entry.Hash)
			if err == nil {
				p.codeWidgets[i].ParseMarkdown("# " + code)
			}
		}
	}
}

func (p *AutoProgressBar) StopAutoUpdate() {
	if p.animation != nil {
		p.animation.Stop()
	}
}

var globalAutoProgress *AutoProgressBar

func main() {
	// Create a new app
	myApp := app.New()
	myWindow := myApp.NewWindow("🔐 My Authenticator")

	// Load entries from encrypted file
	entries, err := decryptAndLoadEntries()
	if err != nil {
		fmt.Println("Error loading entries:", err)
		// Show error dialog and continue with empty entries
		entries = []TOTPEntry{}
	}

	// Create main UI
	createMainUI(myWindow, entries)

	myWindow.ShowAndRun()
}

func createMainUI(myWindow fyne.Window, entries []TOTPEntry) {
	// Create buttons for top row
	// Add entry button with icon only
	addButton := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		showAddEntryDialog(myWindow)
	})
	addButton.Importance = widget.LowImportance

	// Settings button with icon only
	settingsButton := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		dialog.ShowInformation("Settings", "Settings functionality coming soon!", myWindow)
	})
	settingsButton.Importance = widget.LowImportance

	// Export button with icon only
	exportButton := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), func() {
		showExportDialog(myWindow, entries)
	})
	exportButton.Importance = widget.LowImportance

	topButtonRow := container.NewHBox(addButton, exportButton, settingsButton)

	// Create a container to hold all TOTP entries
	var entryWidgets []fyne.CanvasObject
	var codeWidgets []*widget.RichText

	// Create widgets for each TOTP entry
	for i, entry := range entries {
		code, err := generateTOTP(entry.Hash)
		if err != nil {
			fmt.Printf("Error generating TOTP for %s: %v\n", entry.Title, err)
			continue
		}

		if i == 0 {
			entryWidgets = append(entryWidgets, widget.NewSeparator())
		}

		// Create a card-like container for each entry
		titleLabel := widget.NewLabel(entry.Title)
		// titleLabel.TextStyle = fyne.TextStyle{Bold: true}

		codeLabel := widget.NewLabel(code)
		codeLabel.TextStyle = fyne.TextStyle{Monospace: true}

		// Create a rich text widget for bigger TOTP code display
		codeRichText := widget.NewRichTextFromMarkdown("# " + code)
		codeRichText.Wrapping = fyne.TextWrapOff
		codeWidgets = append(codeWidgets, codeRichText) // Store reference for updates

		// Create clickable copy icon for this TOTP code
		copyIcon := widget.NewIcon(theme.ContentCopyIcon())
		copyIcon.Resize(fyne.NewSize(20, 20))

		// Make the icon clickable by creating a button with icon
		copyButton := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			// Capture the current code value
			currentCode := code
			// Get fresh code in case it has been updated
			if freshCode, err := generateTOTP(entry.Hash); err == nil {
				currentCode = freshCode
			}
			// Copy to clipboard using the app
			if app := fyne.CurrentApp(); app != nil {
				app.Clipboard().SetContent(currentCode)
				// Show brief confirmation
				dialog.ShowInformation("Copied", fmt.Sprintf("code copied: %s", currentCode), myWindow)
			}
		})
		copyButton.Resize(fyne.NewSize(24, 24))
		copyButton.Importance = widget.LowImportance // Make it less prominent

		// Create horizontal container with code on left and copy icon right-aligned
		codeWithCopyContainer := container.NewBorder(nil, nil, codeRichText, copyButton, nil)

		// Create entry container with title and code+copy row
		entryContainer := container.NewVBox(
			titleLabel,
			codeWithCopyContainer,
		)

		// Add border/card effect
		entryCard := container.NewBorder(nil, nil, nil, nil, entryContainer)
		entryWidgets = append(entryWidgets, entryCard)

		if i < len(entries)-1 {
			entryWidgets = append(entryWidgets, widget.NewSeparator())
		}
	}

	// If no entries, show a helpful message
	if len(entries) == 0 {
		emptyMessage := widget.NewLabel("No authenticator entries found.\nAdd entries to your encrypted database.")
		emptyMessage.Alignment = fyne.TextAlignCenter
		entryWidgets = append(entryWidgets, emptyMessage)
	}

	// Add single progress bar at bottom showing time remaining for all codes
	now := time.Now()
	remaining := 30 - (now.Second() % 30)

	// Add separator before progress bar
	entryWidgets = append(entryWidgets, widget.NewSeparator())

	// Create progress bar label
	progressLabel := widget.NewLabel(fmt.Sprintf("Next refresh in %d seconds", remaining))
	progressLabel.Alignment = fyne.TextAlignCenter
	entryWidgets = append(entryWidgets, progressLabel)

	// Stop existing auto progress if any
	if globalAutoProgress != nil {
		globalAutoProgress.StopAutoUpdate()
	}

	// Create the single auto-updating progress bar
	globalAutoProgress = NewAutoProgressBar()
	globalAutoProgress.SetValue(float64(remaining) / 30.0)
	globalAutoProgress.SetLabelWidget(progressLabel)     // Connect the label for updates
	globalAutoProgress.SetTOTPData(entries, codeWidgets) // Connect TOTP data for code updates
	globalAutoProgress.StartAutoUpdate()
	entryWidgets = append(entryWidgets, globalAutoProgress)

	// Create scrollable content for entries
	entryContent := container.NewVBox(entryWidgets...)
	scroll := container.NewScroll(entryContent)

	// Create main layout with top button row and scrollable content
	mainContent := container.NewBorder(topButtonRow, nil, nil, nil, scroll)

	myWindow.SetContent(mainContent)
	myWindow.Resize(fyne.NewSize(350, 520))
	myWindow.CenterOnScreen()
}

func showAddEntryDialog(parent fyne.Window) {
	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Enter service name (e.g., Google, GitHub)")

	secretEntry := widget.NewEntry()
	secretEntry.SetPlaceHolder("Enter secret key")
	secretEntry.Password = true

	content := container.NewVBox(
		widget.NewLabel("Service Name:"),
		titleEntry,
		widget.NewLabel("Secret Key:"),
		secretEntry,
	)

	dialog.ShowCustomConfirm("Add New Entry", "Add", "Cancel", content, func(confirmed bool) {
		if !confirmed {
			return
		}

		title := titleEntry.Text
		secret := secretEntry.Text

		if title == "" || secret == "" {
			dialog.ShowError(fmt.Errorf("both service name and secret key are required"), parent)
			return
		}

		// Add new entry
		entries, _ := decryptAndLoadEntries()
		newEntry := TOTPEntry{
			Title: title,
			Hash:  secret,
		}
		entries = append(entries, newEntry)

		// Save entries
		data, _ := json.Marshal(entries)
		err := encryptAndSave(data)
		if err != nil {
			dialog.ShowError(fmt.Errorf("error saving entry: %v", err), parent)
			return
		}

		// Show success message
		dialog.ShowInformation("Success", "New authenticator entry added successfully!", parent)

		// Refresh main UI
		createMainUI(parent, entries)
	}, parent)
}

func showExportDialog(parent fyne.Window, entries []TOTPEntry) {
	if len(entries) == 0 {
		dialog.ShowInformation("Show QR", "No entries to show QR codes for!", parent)
		return
	}

	// Create entry selection
	var entryNames []string
	for _, entry := range entries {
		entryNames = append(entryNames, entry.Title)
	}

	entrySelect := widget.NewSelect(entryNames, nil)
	if len(entryNames) > 0 {
		entrySelect.SetSelected(entryNames[0])
	}

	content := container.NewVBox(
		widget.NewLabel("Select Entry to Show QR Code:"),
		entrySelect,
		widget.NewLabel(""),
		widget.NewLabel("This will display a QR code for the selected entry."),
	)

	dialog.ShowCustomConfirm("Show QR Code", "Show QR", "Cancel", content, func(confirmed bool) {
		if !confirmed {
			return
		}

		selectedTitle := entrySelect.Selected
		if selectedTitle == "" {
			dialog.ShowError(fmt.Errorf("please select an entry"), parent)
			return
		}

		// Find the selected entry
		var selectedEntry *TOTPEntry
		for _, entry := range entries {
			if entry.Title == selectedTitle {
				selectedEntry = &entry
				break
			}
		}

		if selectedEntry == nil {
			dialog.ShowError(fmt.Errorf("selected entry not found"), parent)
			return
		}

		// Show QR code dialog
		showQRCodeDialog(parent, *selectedEntry)
	}, parent)
}

func showQRCodeDialog(parent fyne.Window, entry TOTPEntry) {
	// Create the TOTP URI for QR code
	// Convert hex to base32 if needed
	secret := entry.Hash
	if isHex(secret) {
		base32Secret, err := hexToBase32(secret)
		if err != nil {
			fmt.Printf("Error converting hex to base32: %v\n", err)
			return
		}
		secret = base32Secret
	}

	// Generate TOTP URL for QR code
	otpauthURI := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=TOTP-CLI", entry.Title, secret)

	pngBytes, err := qrcodePNG(otpauthURI, 384)
	if err != nil {
		fmt.Printf("Error creating qrcode PNG: %v\n", err)
		return
	}

	// Create fyne.Resource from PNG bytes and put into canvas image
	res := fyne.NewStaticResource(entry.Title, pngBytes)
	img := canvas.NewImageFromResource(res)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(384, 384))
	// Create instructions
	title := widget.NewLabel(entry.Title)

	content := container.NewVBox(
		title,
		img,
	)
	// Create a custom dialog
	d := dialog.NewCustom("Show QR Code", "Close", content, parent)
	d.Resize(fyne.NewSize(450, 500))
	d.Show()
}

func getKey() []byte {
	password := []byte(encryptionPassword)
	salt := []byte(salt) // store alongside ciphertext
	return pbkdf2.Key(password, salt, 4096, 32, sha256.New)
}
func getGcmAndNonce(key []byte) (cipher.AEAD, []byte) {
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	return gcm, nonce
}
func encryptAndSave(plaintext []byte) error {
	key := getKey()
	gcm, nonce := getGcmAndNonce(key)
	io.ReadFull(rand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return os.WriteFile(dbFileName, ciphertext, 0600)
}

func decryptAndLoadEntries() ([]TOTPEntry, error) {
	var entries []TOTPEntry
	ciphertext, err := os.ReadFile(dbFileName)
	if err != nil {
		fmt.Println("Error reading encrypted file:", err)
		if err.Error() == "open securedata.bin: no such file or directory" {
			encryptAndSave([]byte("[]")) // create empty encrypted file
		}
		return decryptAndLoadEntries()
	}
	gcm, _ := getGcmAndNonce(getKey())
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return entries, err
	}
	json.Unmarshal(plaintext, &entries)
	return entries, nil
}

func generateTOTP(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

// hexToBase32 converts a hex string into base32 (without padding).
func hexToBase32(hexStr string) (string, error) {
	raw, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	// Base32 encode (RFC 4648) and remove padding (=)
	encoded := base32.StdEncoding.EncodeToString(raw)
	encoded = removePadding(encoded)
	return encoded, nil
}

func removePadding(s string) string {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	return s
}

// qrcodePNG returns PNG bytes for the given text at the requested size (pixels)
func qrcodePNG(text string, size int) ([]byte, error) {
	// qrcode.Encode returns PNG bytes
	return qrcode.Encode(text, qrcode.Medium, size)
}
