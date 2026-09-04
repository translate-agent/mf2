# Message Format 2

![CI](https://github.com/expect-digital/go-mf2/actions/workflows/check.yaml/badge.svg)

The repo implements localization by the [Message Format 2](https://github.com/unicode-org/message-format-wg/blob/LDML48.2/spec/message.abnf) specification ([release LDML48.2](https://github.com/unicode-org/message-format-wg/releases/tag/LDML48.2)) of the Message Format Working Group (MFWG):

The project implements the following packages (temporary layout):

- `go.expect.digital/mf2/template` executes MF2 templates (**WIP**)
- `go.expect.digital/mf2/parse` parses MF2 templates
- `go.expect.digital/mf2/builder` builds MF2 templates (**WIP**)
- **CLI** to extract and update localized message strings (**NOT IMPLEMENTED**)

# Requirements

- Golang 1.25+
- IANA Time Zone database - one of:
  - the directory or uncompressed zip file named by the ZONEINFO environment variable
  - on a Unix system, the system standard installation location
  - $GOROOT/lib/time/zoneinfo.zip
  - the time/tzdata package

# Features

## Function registry

List of the default functions registered in the function registry. The functions support localized formatting.

| Function | Signature | Option                                                                                            | Status |
| -------- | --------- | ------------------------------------------------------------------------------------------------- | :----: |
| currency | format    | currency                                                                                          |   ❌   |
| currency | format    | currencyDisplay (narrowSymbol, symbol, name, code, never)                                         |   ❌   |
| currency | format    | currencySign (accounting, standard)                                                               |   ❌   |
| currency | format    | fractionDigits                                                                                    |   ❌   |
| currency | format    | maximumSignificantDigits                                                                          |   ❌   |
| currency | format    | minimumIntegerDigits                                                                              |   ❌   |
| currency | format    | minimumSignificantDigits                                                                          |   ❌   |
| currency | format    | roundingIncrement                                                                                 |   ❌   |
| currency | format    | roundingMode (ceil, floor, expand, trunc, halfCeil, halfFloor, halfExpand, halfTrunc, halfEven)          |   ❌   |
| currency | format    | roundingPriority (auto, morePrecision, lessPrecision)                                             |   ❌   |
| currency | format    | trailingZeroDisplay (auto, stripIfInteger)                                                        |   ❌   |
| currency | format    | useGrouping (auto, always, never, min2)                                                            |   ❌   |
| date     | format    | calendar                                                                                          |   ❌   |
| date     | format    | fields (weekday, day-weekday, month-day, month-day-weekday, year-month-day, year-month-day-weekday) |   ❌   |
| date     | format    | length (long, medium, short)                                                                      |   ❌   |
| date     | format    | timeZone                                                                                          |   ❌   |
| datetime | format    | calendar                                                                                          |   ❌   |
| datetime | format    | dateFields (weekday, day-weekday, month-day, month-day-weekday, year-month-day, year-month-day-weekday) |   ❌   |
| datetime | format    | dateLength (long, medium, short)                                                                  |   ❌   |
| datetime | format    | hour12 (true, false)                                                                              |   ❌   |
| datetime | format    | timePrecision (hour, minute, second)                                                              |   ❌   |
| datetime | format    | timeZone                                                                                          |   ❌   |
| datetime | format    | timeZoneStyle (long, short)                                                                       |   ❌   |
| integer  | format    | maximumSignificantDigits                                                                          |  ✅︎   |
| integer  | format    | minimumIntegerDigits                                                                              |  ✅︎   |
| integer  | format    | signDisplay (auto, always, exceptZero, negative, never)                                           |  ✅︎   |
| integer  | format    | useGrouping (auto, always, never, min2)                                                            |  ✅︎   |
| integer  | match     | maximumSignificantDigits                                                                          |  ✅︎   |
| integer  | match     | minimumIntegerDigits                                                                              |  ✅︎   |
| integer  | match     | select (plural, ordinal, exact)                                                                   |  ✅︎   |
| number   | format    | maximumFractionDigits                                                                             |  ✅︎   |
| number   | format    | maximumSignificantDigits                                                                          |  ✅︎   |
| number   | format    | minimumFractionDigits                                                                             |  ✅︎   |
| number   | format    | minimumIntegerDigits                                                                              |  ✅︎   |
| number   | format    | minimumSignificantDigits                                                                          |   ❌   |
| number   | format    | roundingIncrement                                                                                 |   ❌   |
| number   | format    | roundingMode (ceil, floor, expand, trunc, halfCeil, halfFloor, halfExpand, halfTrunc, halfEven)          |   ❌   |
| number   | format    | roundingPriority (auto, morePrecision, lessPrecision)                                             |   ❌   |
| number   | format    | signDisplay (auto, always, exceptZero, negative, never)                                           |  ✅︎   |
| number   | format    | trailingZeroDisplay (auto, stripIfInteger)                                                        |   ❌   |
| number   | format    | useGrouping (auto, always, never, min2)                                                            |  ✅︎   |
| number   | match     | maximumFractionDigits                                                                             |  ✅︎   |
| number   | match     | maximumSignificantDigits                                                                          |  ✅︎   |
| number   | match     | minimumFractionDigits                                                                             |  ✅︎   |
| number   | match     | minimumIntegerDigits                                                                              |  ✅︎   |
| number   | match     | minimumSignificantDigits                                                                          |   ❌   |
| number   | match     | select (plural, ordinal, exact)                                                                   |  ✅︎   |
| offset   | format    | add                                                                                               |   ❌   |
| offset   | format    | subtract                                                                                          |   ❌   |
| offset   | match     | add                                                                                               |   ❌   |
| offset   | match     | subtract                                                                                          |   ❌   |
| percent  | format    | maximumFractionDigits                                                                             |   ❌   |
| percent  | format    | maximumSignificantDigits                                                                          |   ❌   |
| percent  | format    | minimumFractionDigits                                                                             |   ❌   |
| percent  | format    | minimumSignificantDigits                                                                          |   ❌   |
| percent  | format    | roundingMode (ceil, floor, expand, trunc, halfCeil, halfFloor, halfExpand, halfTrunc, halfEven)          |   ❌   |
| percent  | format    | roundingPriority (auto, morePrecision, lessPrecision)                                             |   ❌   |
| percent  | format    | signDisplay (auto, always, exceptZero, negative, never)                                           |   ❌   |
| percent  | format    | trailingZeroDisplay (auto, stripIfInteger)                                                        |   ❌   |
| percent  | format    | useGrouping (auto, always, never, min2)                                                            |   ❌   |
| percent  | match     |                                                                                                   |   ❌   |
| string   | format    |                                                                                                   |  ✅︎   |
| string   | match     |                                                                                                   |  ✅︎   |
| time     | format    | calendar                                                                                          |   ❌   |
| time     | format    | hour12 (true, false)                                                                              |   ❌   |
| time     | format    | precision (hour, minute, second)                                                                  |   ❌   |
| time     | format    | timeZone                                                                                          |   ❌   |
| time     | format    | timeZoneStyle (long, short)                                                                       |   ❌   |
| unit     | format    | maximumFractionDigits                                                                             |   ❌   |
| unit     | format    | maximumSignificantDigits                                                                          |   ❌   |
| unit     | format    | minimumFractionDigits                                                                             |   ❌   |
| unit     | format    | minimumIntegerDigits                                                                              |   ❌   |
| unit     | format    | minimumSignificantDigits                                                                          |   ❌   |
| unit     | format    | roundingIncrement                                                                                 |   ❌   |
| unit     | format    | roundingMode (ceil, floor, expand, trunc, halfCeil, halfFloor, halfExpand, halfTrunc, halfEven)          |   ❌   |
| unit     | format    | roundingPriority (auto, morePrecision, lessPrecision)                                             |   ❌   |
| unit     | format    | signDisplay (auto, always, exceptZero, negative, never)                                           |   ❌   |
| unit     | format    | unit                                                                                              |   ❌   |
| unit     | format    | unitDisplay (short, narrow, long)                                                                 |   ❌   |
| unit     | format    | usage                                                                                             |   ❌   |
| unit     | format    | useGrouping (auto, always, never, min2)                                                            |   ❌   |
