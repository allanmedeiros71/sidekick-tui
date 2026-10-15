# Calculator

Open it from the menu, or directly with `sidekick calc`. The result updates as
you type; `enter` keeps it in the history, `↑`/`↓` recall earlier inputs and
`esc` goes back to the menu.

For scripts and quick checks, pass the input on the command line:

```sh
sidekick calc '0xff + 1'
sidekick calc 10.0.0.0/22
```

## Integer expressions

64-bit signed integers, shown in decimal, hex, octal and binary.

| Input              | Notes                                                    |
| ------------------ | -------------------------------------------------------- |
| `255`, `1_000_000` | decimal; a leading zero is rejected, use `0o` for octal  |
| `0xFF`, `0b1010`, `0o755` | hex, binary, octal                                |
| `+ - * / % **`     | `/` and `%` truncate toward zero; `**` is power          |
| `& \| ^ ~ << >>`   | bitwise; shifts work on the 64-bit pattern               |
| `( )`              | grouping                                                 |

Precedence, lowest first: `|`, `^`, `&`, `<< >>`, `+ -`, `* / %`, unary
`- + ~`, `**`. As in Python, `-2**2` is `-4` and `**` is right-associative.
Overflow is reported instead of wrapping, except for shifts.

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
- `K M G T P E`: powers of 1024, as in `dd` and `ls -h`

Results that involve a unit are also shown as a human-readable size.
Convert with `in` or `to`: `1.5GiB in MB`, `4096 to KiB`.

## Time

| Input                       | Result                                      |
| --------------------------- | ------------------------------------------- |
| `now`                       | current Unix time                           |
| `today`                     | the start of today (local midnight)         |
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
