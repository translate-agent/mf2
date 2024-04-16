# Message Format 2 Parser

![CI](https://github.com/expect-digital/go-mf2/actions/workflows/ci.yaml/badge.svg)

This parser parses the localized message strings based on the [Message Format 2 Draft](https://github.com/unicode-org/message-format-wg/blob/20a61b4af534acb7ecb68a3812ca0143b34dfc76/spec/message.abnf) by the Message Format Working Group (MFWG).

# Requirements

- Golang 1.22+
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
| datetime               | format    | calendar                                      |   ❌   |
| datetime               | format    | numberingSystem                               |   ❌   |
| datetime               | format    | timeZone                                      |   ❌   |
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
| number                 | format    | currency                                      |   ❌   |
| number                 | format    | currencyDisplay                               |   ❌   |
| number                 | format    | currencySign                                  |   ❌   |
| number                 | format    | notation                                      |   ❌   |
| number                 | format    | numberingSystem                               |   ❌   |
| number                 | format    | signDisplay (auto, always, exceptZero, never) |  ✅︎   |
| number                 | format    | style (decimal, percent)                      |  ✅︎   |
| number                 | format    | style (currency, unit)                        |   ❌   |
| number                 | format    | unit                                          |   ❌   |
| number                 | format    | unitDisplay                                   |   ❌   |
| number                 | format    | minimumIntegerDigits                          |   ❌   |
| number                 | format    | minimumFractionDigits                         |  ✅︎   |
| number                 | format    | maximumFractionDigits                         |  ✅︎   |
| number                 | format    | minimumSignificantDigits                      |   ❌   |
| number                 | format    | maximumSignificantDigits                      |   ❌   |
| number                 | match     | select                                        |   ❌   |
| number                 | match     | minimumIntegerDigits                          |   ❌   |
| number                 | match     | minimumFractionDigits                         |   ❌   |
| number                 | match     | maximumFractionDigits                         |   ❌   |
| number                 | match     | minimumSignificantDigits                      |   ❌   |
| number                 | match     | maximumSignificantDigits                      |   ❌   |
| number                 | match     | minimumFractionDigits                         |   ❌   |
| number                 | match     | minimumFractionDigits                         |   ❌   |
| integer (number alias) |           |                                               |   ❌   |
| ordinal (number alias) |           |                                               |   ❌   |
| plural (number alias)  |           |                                               |   ❌   |
| string                 |           |                                               |   ❌   |
| time                   |           |                                               |   ❌   |
