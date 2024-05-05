/*
Package utils implements helpful funcs that are reused throughout the app
and provide some simply, yet redundantly coded, functionaly.  Think of this
package as storing some "generic" funcs that golang doesn't provide or funcs
that we reuse in a bunch of places.

This package should not import any other packages.  This package should only
be imported into other packages.  This is to prevent import loops.  Plus,
considering that these are basic helper funcs there should be no need to depend
on anything else (maybe the std lib).
*/
package utils
