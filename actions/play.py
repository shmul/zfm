import pydub
import pydub.playback
import pathlib


def playall(head: int, tail: int, files: str):
    for f in files:
        audio = pydub.AudioSegment.from_file(pathlib.Path(f))
        ln = len(audio)
        if head == 0 and tail == 0:
            pydub.playback.play(audio)
            continue

        if head > 0:
            pydub.playback.play(audio[:head])

        if tail > 0:
            pydub.playback.play(audio[ln - tail:])
