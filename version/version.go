//Package version store's the app's version for use in diagnostics.
package version

//V is the version number of the app. This should match the git tag at the
//point this version was released. This value is stored here, and not in main.go,
//so that we can get it from any other package as needed (aka pages for diagnostic
//page).
const V = "2.0.0"
