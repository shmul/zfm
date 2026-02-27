import actions.play
import actions.crop
import actions.generate
import click

@click.group()
def zfm():
    pass


@zfm.command()
@click.option('--start',
              '-s',
              type=str,
              help='start position (format [hh:]mm:ss)',
              show_default=True)
@click.option('--end',
              '-e',
              type=str,
              help='start position (format [hh:]mm:ss)',
              show_default=True)
@click.option('--head',
              '-h',
              default=0,
              type=float,
              help='head position (in secs)',
              show_default=True)
@click.option('--tail',
              '-t',
              default=0,
              type=float,
              help='tail position (in secs from end of the track)',
              show_default=True)
@click.option('--fade-in',
              '-i',
              default=0,
              type=float,
              help='fade in duration (in secs)',
              show_default=True)
@click.option('--fade-out',
              '-o',
              default=0,
              type=float,
              help='fade in duration (in secs)',
              show_default=True)
@click.option('--play/--no-play',
              '-p',
              default=False,
              help='play instead of save',
              show_default=True)
@click.option('--target-dir',
              default='',
              help='target dir for cropped files',
              show_default=False)
@click.option('--dry-run/--no-dry-run',
              '-n',
              default=False,
              help='just prepare but don\'t write',
              show_default=True)
@click.option('--analyze-silence/--no-analyze-silence',
              '-a',
              default=False,
              help='analyze tail for silence detection',
              show_default=True)
@click.option('--silence-thresh',
              default=-40.0,
              type=float,
              help='silence threshold in dBFS',
              show_default=True)
@click.argument('filename', type=click.Path(exists=True))
def crop(start, end, head, tail, fade_in, fade_out, play, target_dir, dry_run,
         analyze_silence, silence_thresh, filename):
    '''Crop file'''
    click.echo('zfm crop')
    actions.crop.crop(filename, start, end, head, tail, fade_in, fade_out,
                      play, target_dir, dry_run, analyze_silence, silence_thresh)


@zfm.command()
@click.option('--target-dir',
              default='',
              help='target dir for cropped files',
              show_default=False)
@click.option('--preview',
              '-p',
              default=0,
              type=float,
              help='play preview (secs) of head and end',
              show_default=True)
@click.option('--just',
              '-j',
              default=-1,
              type=int,
              help='play just that audio (zero based indexing)',
              show_default=True)
@click.option('--one-by-one',
              '-1',
              is_flag=True,
              default=False,
              type=bool,
              help='play tracks individually',
              show_default=True)
@click.option('--dry-run/--no-dry-run',
              '-n',
              default=False,
              help='just prepare but don\'t write',
              show_default=True)
@click.argument('filename', type=click.Path(exists=True))
def csv(filename, target_dir, preview, just, one_by_one, dry_run):
    '''Prepare from csv'''
    actions.crop.crop_many(filename, target_dir, preview, just, one_by_one,
                           dry_run)


@zfm.command()
@click.option('--target-dir',
              default='',
              help='target dir for csv file',
              show_default=False)
@click.argument('filename', type=click.Path(exists=True))
def m3u(filename, target_dir):
    '''m3u to csv'''
    actions.crop.to_csv(filename, target_dir)


@zfm.command()
@click.option('--head',
              '-h',
              default=0,
              type=float,
              help='head position (in secs)',
              show_default=True)
@click.option('--tail',
              '-t',
              default=0,
              type=float,
              help='tail position (in secs from end of the track)',
              show_default=True)
@click.option('--fade-in',
              '-i',
              default=0,
              type=float,
              help='fade in duration (in secs)',
              show_default=True)
@click.option('--fade-out',
              '-o',
              default=0,
              type=float,
              help='fade in duration (in secs)',
              show_default=True)
@click.argument('files', nargs=-1)
def play(head, tail, fade_in, fade_out, files):
    '''Play files'''
    actions.play.playall(head, tail, fade_in, fade_out, files)


@zfm.command()
@click.argument('dir', nargs=1)
def generate(dir):
    '''generate playlist.csv file from dir'''
    actions.generate.generate(dir)


@zfm.command()
@click.option('--analysis-duration',
              '-d',
              default=30.0,
              type=float,
              help='seconds of audio tail to analyze',
              show_default=True)
@click.option('--silence-thresh',
              '-t',
              default=-40.0,
              type=float,
              help='silence threshold in dBFS',
              show_default=True)
@click.option('--min-silence',
              '-m',
              default=1.0,
              type=float,
              help='minimum silence duration in seconds',
              show_default=True)
@click.argument('filename', type=click.Path(exists=True))
def analyze(analysis_duration, silence_thresh, min_silence, filename):
    '''analyze audio file for silence detection'''
    import pydub
    from actions.volume import analyze_tail_silence, format_silence_analysis
    
    print(f"Loading: {filename}")
    audio = pydub.AudioSegment.from_file(filename)
    
    analysis = analyze_tail_silence(
        audio, 
        analysis_duration=analysis_duration,
        silence_thresh=silence_thresh,
        min_silence_len=min_silence
    )
    
    print(format_silence_analysis(analysis))

if __name__ == '__main__':
    zfm()
