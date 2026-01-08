package wa

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

const DEFAULT_IMAGE = "data:image/svg+xml;base64,PHN2ZyAgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIiAgd2lkdGg9IjI0IiAgaGVpZ2h0PSIyNCIgIHZpZXdCb3g9IjAgMCAyNCAyNCIgIGZpbGw9Im5vbmUiICBzdHJva2U9ImN1cnJlbnRDb2xvciIgIHN0cm9rZS13aWR0aD0iMiIgIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIgIHN0cm9rZS1saW5lam9pbj0icm91bmQiICBjbGFzcz0iaWNvbiBpY29uLXRhYmxlciBpY29ucy10YWJsZXItb3V0bGluZSBpY29uLXRhYmxlci1rZXkiPjxwYXRoIHN0cm9rZT0ibm9uZSIgZD0iTTAgMGgyNHYyNEgweiIgZmlsbD0ibm9uZSIvPjxwYXRoIGQ9Ik0xNi41NTUgMy44NDNsMy42MDIgMy42MDJhMi44NzcgMi44NzcgMCAwIDEgMCA0LjA2OWwtMi42NDMgMi42NDNhMi44NzcgMi44NzcgMCAwIDEgLTQuMDY5IDBsLS4zMDEgLS4zMDFsLTYuNTU4IDYuNTU4YTIgMiAwIDAgMSAtMS4yMzkgLjU3OGwtLjE3NSAuMDA4aC0xLjE3MmExIDEgMCAwIDEgLS45OTMgLS44ODNsLS4wMDcgLS4xMTd2LTEuMTcyYTIgMiAwIDAgMSAuNDY3IC0xLjI4NGwuMTE5IC0uMTNsLjQxNCAtLjQxNGgydi0yaDJ2LTJsMi4xNDQgLTIuMTQ0bC0uMzAxIC0uMzAxYTIuODc3IDIuODc3IDAgMCAxIDAgLTQuMDY5bDIuNjQzIC0yLjY0M2EyLjg3NyAyLjg3NyAwIDAgMSA0LjA2OSAweiIgLz48cGF0aCBkPSJNMTUgOWguMDEiIC8+PC9zdmc+"

func (wa *WA) PasskeyImageHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["aaguid"]

	imageString := DEFAULT_IMAGE
	if entry, ok := wa.aaguidMap[id]; ok {
		imageString = entry.IconLight
	}

	parts := strings.Split(imageString, ",")
	if len(parts) != 2 {
		http.Error(w, "Invalid base64 string", http.StatusInternalServerError)
		return
	}

	base64Data := parts[1]

	// Decode the base64 string to a byte slice.
	decodedImage, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		http.Error(w, "Could not decode base64", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(decodedImage)
}
