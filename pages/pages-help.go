/*
Package pages is used to display the gui. This package is kind of a middleman for
other funcs and the templates package.

This file specifically deals with help/documentation pages. As long as the filename
matches what is used in the <a> tag on the table of contents, the Help func will
serve the file.
*/
package pages

import (
	"net/http"

	"github.com/c9845/templates"
	"github.com/gorilla/mux"
)

// HelpTableOfContents shows the page listing help documents
func HelpTableOfContents(w http.ResponseWriter, r *http.Request) {
	//show page
	templates.Show(w, "help", "table-of-contents", nil)
}

// Help shows a help documentation page. This serves any help page by using the end of
// the url as the document name. This is useful since we don't need to create a func
// or route for each help page we create.
func Help(w http.ResponseWriter, r *http.Request) {
	//get data from url
	vars := mux.Vars(r)
	doc := vars["document"]

	//show page
	templates.Show(w, "help", doc, nil)
}
