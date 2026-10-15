# Calculator

Open it from the menu, or directly with `sidekick calc`. The result updates as
you type; `enter` keeps it in the history, `↑`/`↓` recall earlier inputs and
`esc` goes back to the menu.

For scripts and quick checks, pass the input on the command line or pipe it
in:

```sh
sidekick calc '0xff + 1'
sidekick calc 10.0.0.0/22
echo '2**32' | sidekick calc
```

## Expressions

Arithmetic is exact: `0.1 + 0.2` is `0.3` and `10 / 3` keeps every digit.
Whole numbers that fit in 64 bits are shown in decimal, hex, octal and
binary; other results are shown as a decimal, as a fraction when it is
short (`10/3`), and in scientific notation (`3.33333333333333e+00`).

| Input              | Notes                                                    |
| ------------------ | -------------------------------------------------------- |
| `255`, `1_000_000`, `2.5` | decimal; a leading zero is rejected, use `0o` for octal |
| `6.022e23`, `1.6e-19` | scientific notation                                   |
| `0xFF`, `0b1010`, `0o755` | hex, binary, octal                                |
| `+ - * /`          | exact; `/` gives a fraction (`7 / 2` is `3.5`)           |
| `//`, `%`          | whole-number division and remainder, truncating toward zero like C |
| `**`               | power; exact for whole exponents, approximate otherwise (`2 ** 0.5`) |
| `& \| ^ ~ << >>`   | bitwise, on 64-bit whole numbers; shifts work on the bit pattern |
| `( )`              | grouping                                                 |

Precedence, lowest first: `|`, `^`, `&`, `<< >>`, `+ -`, `* / // %`, unary
`- + ~`, `**`. As in Python, `-2**2` is `-4` and `**` is right-associative.
Decimals show up to 15 significant digits. Very large or very small values
are shown in scientific notation, and whole numbers of 13 digits or more
show it too.

Hex, binary and octal literals of up to 64 bits are bit patterns, so
`0xFFFFFFFFFFFFFFFF` is `-1`. Negative results also show their unsigned value.
Values between 32 and 126 show their ASCII character, and values that look
like recent Unix timestamps (seconds or milliseconds) show the date.

## Sizes

A number can carry a size unit, with or without a space: `4KiB`, `1.5 GB`,
`512M * 3`. Units are case-insensitive, except that a bare `m` means
minutes, so write `M` for MiB:

- `B`
- `KiB MiB GiB TiB PiB EiB`: powers of 1024
- `KB MB GB TB PB EB`: powers of 1000
- `K M G T P E`: powers of 1024, as in `dd` and `ls -h` (`2E` is 2 EiB,
  while `2e3` is 2000)

Results that involve a unit are also shown as a human-readable size.
Convert with `in` or `to`: `1.5GiB in MB`, `4096 to KiB`.

## Time

| Input                       | Result                                      |
| --------------------------- | ------------------------------------------- |
| `now`                       | current Unix time                           |
| `today`                     | the start of today (local midnight)         |
| `age 1971-04-30`            | age today, e.g. `55 years 5 months 15 days`, with the next birthday and days lived |
| `@1700000000`               | timestamp to date; the unit (s, ms, µs, ns) is guessed from the magnitude |
| `2024-05-01`, `2024-05-01 13:30`, RFC 3339 | date to timestamp; dates without a zone use the local zone |

### Date arithmetic

Add or subtract durations from a date (`now`, `today`, a date as above, or
`@timestamp`), or subtract two dates to get the time between them. Use
`today` for whole days: it has no time of day, so `today + 3mo` lands on a
date and `2026-12-25 - today` counts whole days. Results show the date with
its weekday, and dates at midnight are described in calendar days
(`tomorrow`, `in 3 days`).

| Input                  | Result                                         |
| ---------------------- | ---------------------------------------------- |
| `now + 3d`             | the date three days from now                   |
| `today + 3mo`          | the date three months from today               |
| `2026-01-01 - 2w`      | two weeks before New Year                      |
| `@1700000000 + 1h30m`  | a timestamp plus 90 minutes                    |
| `2026-12-25 - now`     | time left, e.g. `70 days (10 weeks) 12 hours`, also in seconds, hours, days and weeks |

Duration units are `y` (years), `mo` (months), `w` (weeks), `d` (days), `h`,
`m` (minutes) and `s`, and can be combined: `1y2mo`, `1h30m`. Years, months
and days follow the calendar: the day of the month is kept and clamped to the
month's end, so `2026-01-31 + 1mo` is `2026-02-28` and `2024-02-29 + 1y` is
`2025-02-28`, and a day across a DST change keeps the clock time.

### Durations

A duration on its own, such as `30d` or `1h30m`, is shown in seconds,
minutes, hours, days and weeks. Convert one with `in` or `to`: `30d in w`,
`90m in h`, `1y to d`, `6mo in w`. Without a date there is no calendar to
follow, so a year counts as 365 days and a month as a twelfth of that.

## chmod

`chmod 755`, `chmod 4755`, `rwxr-xr-x` or `-rw-r--r--` (as printed by
`ls -l`) show the mode in octal and symbolic form, what each class may do and
any setuid, setgid or sticky bit.

## Networks

- `10.0.0.0/22`: network, netmask, wildcard, broadcast, host range and counts.
  `/31` and `/32` follow RFC 3021. If host bits are set, the containing network
  is shown.
- `2001:db8::/48`: first and last address and the address count.
- `192.168.0.10`: the address as an integer, in hex and binary, and its scope
  (private, loopback, global...).
