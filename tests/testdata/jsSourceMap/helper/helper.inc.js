// Helper JavaScript code for jsSourceMap tests that gets the stack trace
// so that the stack frames can be checked for expected source mapping.
function doJSThing() {
    return Error().stack;
}

// Give access to the function from Go code via js.Global.
this.doJSThing = doJSThing;
