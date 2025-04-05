# Message Format 2

![CI](https://github.com/expect-digital/go-mf2/actions/workflows/ci.yaml/badge.svg)

The repo implements localization by the [Message Format 2 Draft](https://github.com/unicode-org/message-format-wg/blob/b4fd5a666a02950c57f0a454f65bf16a0bf03bf4/spec/message.abnf) of the Message Format Working Group (MFWG):

The project implements the following packages (temporary layout):

- `go.expect.digital/mf2/template` executes MF2 templates (**WIP**)
- `go.expect.digital/mf2/parse` parses MF2 templates (**WIP**)
- `go.expect.digital/mf2/builder` builds MF2 templates (**WIP**)
- **CLI** to extract and update localized message strings (**NOT IMPLEMENTED**)

# Requirements

- Golang 1.23+
- IANA Time Zone database - one of:
  - the directory or uncompressed zip file named by the ZONEINFO environment variable
  - on a Unix system, the system standard installation location
  - $GOROOT/lib/time/zoneinfo.zip
  - the time/tzdata package

# Features

## Function registry

List of the default functions registered in the function registry. The functions support localized formatting.

| Function               | Signature | Option                                        | Status |
| ---------------------- | --------- | --------------------------------------------- | :----: |
| date                   | format    | style                                         |   ❌   |
| datetime               | format    | dateStyle                                     |   ❌   |
| datetime               | format    | timeStyle                                     |   ❌   |
| datetime               | format    | timeZone<sup>\*</sup>                         |   ❌   |
| datetime               | format    | hourCycle                                     |   ❌   |
| datetime               | format    | dayPeriod                                     |   ❌   |
| datetime               | format    | weekday                                       |   ❌   |
| datetime               | format    | era                                           |   ❌   |
| datetime               | format    | year                                          |   ❌   |
| datetime               | format    | month                                         |   ❌   |
| datetime               | format    | day                                           |   ❌   |
| datetime               | format    | hour                                          |   ❌   |
| datetime               | format    | minute                                        |   ❌   |
| datetime               | format    | second                                        |   ❌   |
| datetime               | format    | fractionalSecondDigits                        |   ❌   |
| datetime               | format    | timeZoneName                                  |   ❌   |
| number                 | format    | compactDisplay                                |   ❌   |
| number                 | format    | currency<sup>\*</sup>                         |   ❌   |
| number                 | format    | currencyDisplay<sup>\*</sup>                  |   ❌   |
| number                 | format    | currencySign<sup>\*</sup>                     |   ❌   |
| number                 | format    | notation                                      |   ❌   |
| number                 | format    | numberingSystem                               |   ❌   |
| number                 | format    | signDisplay (auto, always, exceptZero, never) |  ✅︎   |
| number                 | format    | style (decimal, percent)                      |  ✅︎   |
| number                 | format    | style (currency, unit)                        |   ❌   |
| number                 | format    | unit<sup>\*</sup>                             |   ❌   |
| number                 | format    | unitDisplay<sup>\*</sup>                      |   ❌   |
| number                 | format    | minimumIntegerDigits                          |  ✅︎   |
| number                 | format    | minimumFractionDigits                         |  ✅︎   |
| number                 | format    | maximumFractionDigits                         |  ✅︎   |
| number                 | format    | minimumSignificantDigits                      |   ❌   |
| number                 | format    | maximumSignificantDigits                      |  ✅︎   |
| number                 | match     | select                                        |  ✅︎   |
| number                 | match     | minimumIntegerDigits                          |  ✅︎   |
| number                 | match     | minimumFractionDigits                         |  ✅︎   |
| number                 | match     | maximumFractionDigits                         |  ✅︎   |
| number                 | match     | minimumSignificantDigits                      |   ❌   |
| number                 | match     | maximumSignificantDigits                      |  ✅︎   |
| integer (number alias) | format    |                                               |  ✅︎   |
| integer (number alias) | match     |                                               |  ✅︎   |
| ordinal (number alias) |           |                                               |   ❌   |
| plural (number alias)  |           |                                               |   ❌   |
| string                 |           |                                               |  ✅︎   |
| time                   | format    | style                                         |   ❌   |

> **<sup>\*</sup>** The options are not part of the default registry. MF2 WG says, "Implementations SHOULD avoid creating options that conflict with these, but are encouraged to track development of these options during Tech Preview".
