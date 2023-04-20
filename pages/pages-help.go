package pages

import (
	"net/http"

	"github.com/gorilla/mux"
)

//This file handles displaying the in-app help documentation.

// HelpTableOfContents shows the page listing help documents
func HelpTableOfContents(w http.ResponseWriter, r *http.Request) {
	//Cache HTML in the browser since it rarely changes (app updates). This just makes
	//the GUI load a tiny bit quicker. Data for pages is loaded via API calls so this
	//just caches the base HTML.
	setBaseHTMLCacheHeaders(w)

	//show page
	Show(w, "help", "table-of-contents", nil)
}

// Help shows a help documentation page. This serves any help page by using the end of
// the url as the document name. This is useful since we don't need to create a func
// or route for each help page we create.
func Help(w http.ResponseWriter, r *http.Request) {
	//Cache HTML in the browser since it rarely changes (app updates). This just makes
	//the GUI load a tiny bit quicker. Data for pages is loaded via API calls so this
	//just caches the base HTML.
	setBaseHTMLCacheHeaders(w)

	//get data from url
	vars := mux.Vars(r)
	doc := vars["document"]

	//show page
	Show(w, "help", doc, nil)
}
