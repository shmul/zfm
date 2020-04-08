import click

@click.group()
def zfm():
    pass

@zfm.command()
def cmd1():
    '''Command on zfm'''
    click.echo('zfm cmd1')

@zfm.command()
def cmd2():
    '''Command on zfm'''
    click.echo('zfm cmd2')

if __name__ == '__main__':
    zfm()
