import typing
import sys
import pathlib
from os import walk
from pathlib import Path
from actions.crop import *

def generate(dir: str):
    with open(os.path.join(dir,"playlist.csv"),'w') as f:
        print(",".join(csv_field_names),file=f)

        for (dirpath, dirnames, filenames) in walk(dir):
            for name in filenames:
                if name.endswith(".mp3") or name.endswith(".flac"):
                    print( "\"{}\",".format(os.path.join(dir,name)),file=f)
