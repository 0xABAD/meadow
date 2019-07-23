package main

const FAVICON = `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`
