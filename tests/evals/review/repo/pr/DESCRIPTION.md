# feat: back up notes to your server

Adds `jotter.py backup`. It uploads every note to the server in `config.json`.

- Notes and folders listed in `.backupignore` never leave the machine.
- The token stays secret: the log names only the note being uploaded.
- The summary counts only the notes the server accepted.
- Tested: `python3 -m unittest -v` passes.
