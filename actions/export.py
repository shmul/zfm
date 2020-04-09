import pydub
import typing
import pathlib


#AudioFiles = typing.NewType('AudioFiles', List[string])
def export(audios: AudioFiles, dest: str, format="mp3", bitrate="320k"):
    playlist = AudioSegment.empty()
    for audio in audios:
        playlist += audio

    return playlist.export(dest, format, bitrate)
