# Universalconverter/Uniconvert
A universal compressor, converter, and encryptor for Mac! Designed for #Horizons.
This converter is a simple TUI (terminal UI) app that allows you to convert, compress, and encrypt files on Mac, right from the terminal. No need for websites or tools that have a monthly subscription. i really made this one just because when i needed to convert something and i had a spotty connection, it was prettty annoying, and no more dealing with size limits too, and the risk of privacy breach too lol.

note to horizion reviewers: i made this app in like 3 hours (5 hours including planning) so i didnt make the repo first, thats an issue on my end as i thought that i couldn't really add incremental push progress to it so please pardon me for pushing it all at once.

using the simple command of: ``` uniconvert ```


# Stack
Uniconvert is made in Go, using the **charm.sh TUI framework** (`bubbletea`, `lipgloss`, `bubbles`).

# Feature list
There are 6 features in this TUI. Here's a quick rundown of them:

# **Feature one: Compression**
Pack files with zstd, brotli, gzip, lzma, and standard zip/tar archives, fast!
(Psst, I included drag n' drop so you don't have to copy file paths.)
<img width="1022" height="569" alt="Screenshot 2026-07-03 at 6 42 48 PM" src="https://github.com/user-attachments/assets/265f0003-af28-4682-b823-ebcd36683a52" />


# **Feature two: Encryption**
Secure files with military-grade AES-256-GCM or ChaCha20-Poly1305 using Argon2id key derivation. (I don't really know how these work.)
<img width="1095" height="676" alt="Screenshot 2026-07-03 at 7 37 19 PM" src="https://github.com/user-attachments/assets/b031309d-1607-48e6-be66-f79570e142d5" />


# **Feature three: Data formats (broken)**
This one is currently broken, but here's what it does: it allows you to convert hierarchical formats like JSON, YAML, TOML, XML, and tabular CSV files.
(I will be fixing this one soon, but I can't seem to find the issue with the keyboard-based interaction.)
<img width="1092" height="687" alt="Screenshot 2026-07-03 at 7 20 17 PM" src="https://github.com/user-attachments/assets/5ceaebfa-4dbf-4a72-8a6f-50fb9ad06904" />


# **Feature four: Media Convert**
As the name suggests, it allows you to perform format swaps for videos, audio tracks, and images, or extract audio tracks
<img width="1097" height="684" alt="Screenshot 2026-07-03 at 7 20 28 PM" src="https://github.com/user-attachments/assets/b7ca4a08-7cf9-4f0b-a12d-5b79303e0586" />


# **Feature five: Text & hash**
This feature allows you to compute cryptographic hashes and text representations of strings or local files.
<img width="1101" height="687" alt="Screenshot 2026-07-03 at 7 20 44 PM" src="https://github.com/user-attachments/assets/95bbf12a-e140-4303-8be2-350b721d0161" />


# **Feature six: Unit converter**
This one is pretty simple, and it allows you to convert units to another unit.
<img width="1091" height="684" alt="Screenshot 2026-07-03 at 7 20 55 PM 1" src="https://github.com/user-attachments/assets/af15cadf-e1f7-4d62-a6a8-1c76c42a3a21" />

## Installation

### Requirements

- macOS
- Go 1.20+
- Homebrew
- FFmpeg (required for the Media Converter)

Install FFmpeg:

```bash
brew install ffmpeg
```

### Build

Clone the repository, then run:

```bash
go mod tidy
go build -o uniconvert main.go
```

Launch the app:

```bash
uniconvert
```
# Contributing

Pull requests are welcome!

If you discover a bug or have an idea, feel free to open an issue.

# License

MIT License

# Potential Future

Here's what I'd like to add as the project evolves.

- Plugin system for community-made converters
- Batch processing and queue support
- Folder watch mode for automatic conversions
- Better archive management (preview without extraction)
- PDF utilities (merge, split, compress)
- Image optimization (WebP, AVIF, metadata stripping)
- Built-in benchmark mode for compression algorithms
- Theme support and customization
- Linux support
- Windows support

funny note: i had to use apple emojis cause i wasn't able to use phosphor icons

made with <3 by me! for hackclub horizions.
