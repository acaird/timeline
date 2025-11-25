
The [PEG](https://en.wikipedia.org/wiki/Parsing_expression_grammar) file here - `timeline.peg` is input to [Pigeon](https://github.com/mna/pigeon). It is intended to parse Wikipedia's [timeline format](https://en.wikipedia.org/wiki/Help:EasyTimeline_syntax).

The `_test.go` file let's you confirm that the parser is working without writing a real program. Eventually it will grow up to be a real set of tests.

## Running

```
go install github.com/mna/pigeon@latest
pigeon -o parser.go timeline.peg
go test -v .
```
and, depending on how in-sync this document is with the other files, you may see:
```
=== RUN   TestParse
{Width:945 Height:auto Barincrement:20}
--- PASS: TestParse (0.00s)
PASS
ok      github.com/acaird/timeline/peg  0.380s
```


## To Do

   - [ ] the rest of the grammar
   - [ ] tests for this parser
   - [ ] replace the other parser with this one
