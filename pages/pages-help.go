package pages

import (
	"net/http"
)

//This file handles displaying the in-app help documentation.
//This file handles displaying the in-app help documentation.

// HelpTableOfContents shows the page listing help documents
func HelpTableOfContents(w http.ResponseWriter, r *http.Request) {
	//show page
	Show(w, "help/table-of-contents.html", nil)
}
