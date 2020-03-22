package main

import (
	"bytes"
	"fmt"
	"html/template"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/jung-kurt/gofpdf"
	pdfbc "github.com/jung-kurt/gofpdf/contrib/barcode"
)

const (
	baseStorePattern       = "/store/"
	projectDomain          = "engpass.appspot.com"
	placeIDFormat          = "[A-Za-z0-9-_]+"
	whatsLeftDomain        = "whatsleft.wirvsvirus.net"
	qrErrorCorrectionLevel = qr.H
)

type store struct {
	GooglePlaceID string
	WhatsLeftURL  string
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc(baseStorePattern, storeHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func storeHandler(w http.ResponseWriter, r *http.Request) {
	validPath := regexp.MustCompile("^" + baseStorePattern + "(" + placeIDFormat + ")(/(pdf|png|qr))?$")

	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)

		return
	}

	googlePlaceID := m[1]
	switch m[3] {
	case "":
		targetURL := whatsLeftStoreURL(googlePlaceID)
		log.Printf("weiterleiten zu %s ...", targetURL)

		w.Header().Set("Location", targetURL)
		w.WriteHeader(302)

		fp := path.Join("assets/templates", "store-whatsleft.html")
		tmpl, err := template.ParseFiles(fp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, store{
			GooglePlaceID: googlePlaceID,
			WhatsLeftURL:  targetURL,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		break
	case "pdf":
		targetURL := projectStoreURL(googlePlaceID)
		title := "WhatsLeft - " + googlePlaceID
		fileName := strings.ReplaceAll(strings.ToLower(title), " ", "_") + ".pdf"

		log.Printf("generiere PDF %s mit QR-Code der auf %s zeigt", fileName, targetURL)

		var buf bytes.Buffer

		if err := generatePDF(&buf, targetURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileName))
		w.Write(buf.Bytes())

		break
	case "qr":
		targetURL := projectStoreURL(googlePlaceID)
		log.Printf("generiere QR-Code der auf %s zeigt", targetURL)
		if err := generateQRCode(w, targetURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		break
	default:
		http.NotFound(w, r)
	}

	return
}

func generatePDF(w io.Writer, targetURL string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	qr := pdfbc.RegisterQR(pdf, targetURL, qrErrorCorrectionLevel, qr.Auto)
	qrSize := float64(120) // mm
	qrBorder := float64((210 - qrSize) / 2)
	pdfbc.Barcode(pdf, qr, qrBorder, 279-20-qrSize, qrSize, qrSize, false)

	pdf.SetX(60)
	pdf.ImageOptions(
		"assets/images/teaser-whatsleft.png",
		20,  // x
		20,  // y
		170, // w
		0,   // h = auto
		false,
		gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   true,
		},
		0,
		"")

	pdf.ImageOptions(
		"assets/images/logo-projekt.png",
		120, // x
		267, // y
		70,  // w
		0,   // h = auto
		false,
		gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   true,
		},
		0,
		"")

	return pdf.Output(w)
}

func generateQRCode(writer io.Writer, targetURL string) error {
	qrCode, _ := qr.Encode(targetURL, qrErrorCorrectionLevel, qr.Auto)
	qrCode, _ = barcode.Scale(qrCode, 1024, 1024)

	return png.Encode(writer, qrCode)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("handle URL path %s ...", r.URL.Path)

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprint(w, "Hello world!")
}

func projectStoreURL(googlePlaceID string) string {
	return "https://" + projectDomain + baseStorePattern + googlePlaceID
}

func whatsLeftStoreURL(googlePlaceID string) string {
	return "https://" + whatsLeftDomain + "/store/" + googlePlaceID
}
