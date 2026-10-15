# Calendar

Open it from the menu, or directly with `sidekick cal`. It opens on today's
month, with weeks starting on Monday and each row led by its ISO 8601 week
number. Today is underlined and the selected day is highlighted; the line
under the grid gives the selected day's weekday, ISO week, day of the year
and how far it is from today.

| Key                    | Does                                       |
| ---------------------- | ------------------------------------------ |
| `←` `→` (`h` `l`)      | previous / next day                        |
| `↑` `↓` (`k` `j`)      | previous / next week                       |
| `pgup` `pgdn` (`<` `>`)| previous / next month                      |
| `[` `]`                | previous / next year                       |
| `t`                    | back to today                              |
| `g` or `/`             | go to a date (see below); `esc` cancels    |
| `enter`                | copy the selected date as `YYYY-MM-DD`     |
| `esc`                  | back to the menu                           |

Moving by months or years keeps the day of the month, clamped to the
month's last day (Jan 31 moves to Feb 28).

## Going to a date

`g` opens a prompt that takes:

| Input                         | Goes to                                      |
| ----------------------------- | -------------------------------------------- |
| `2026-12-25`, `today`, `@1700000000` | that date                              |
| `2026-12`                     | the 1st of that month                        |
| `2027`                        | January 1st of that year                     |
| `today + 90d`, `2026-01-31 + 1mo` | the date, using the [calculator's date arithmetic](calculator.md#date-arithmetic) |
| `+2w`, `-45d`, `+1y`          | that far from the selected day               |

## Without the UI

`sidekick cal <date>` prints the month holding the date and a line
describing it, taking the same inputs as the prompt:

```sh
$ sidekick cal today + 90d
      January 2027
wk  Mo Tu We Th Fr Sa Su
53               1  2  3
 1   4  5  6  7  8  9 10
 2  11 12 13 14 15 16 17
 3  18 19 20 21 22 23 24
 4  25 26 27 28 29 30 31

Wednesday 2027-01-13 · week 2 · day 13 of 365 · in 90 days
```

`cal` and `ncal` already print months; this module is for when sidekick is
already open, for week numbers, and for date arithmetic without leaving the
popup.
