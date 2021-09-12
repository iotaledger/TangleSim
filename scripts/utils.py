"""The helper functions.
"""

import numpy as np
import argparse
import logging
import os


class ArgumentParserWithDefaults(argparse.ArgumentParser):
    """The argument parser to support RawTextHelpFormatter and show default values.
    """

    def add_argument(self, *args, help=None, default=None, **kwargs):
        if help is not None:
            kwargs['help'] = help
        if default is not None and args[0] != '-h':
            kwargs['default'] = default
            if help is not None:
                kwargs['help'] += '\nDefault: {}'.format(default)
        super().add_argument(*args, **kwargs)


def move_results(src, dst):
    """Move the files from the source folder to the destination folder.

    Args:
        src: The source folder.
        dst: The destination folder.
    """

    if not os.path.isdir(dst):
        os.mkdir(dst)
    logging.info(f'Moving folder {src} to {dst}...')
    os.system(f'mv {src}/*.config {dst}')
    os.system(f'mv {src}/*.csv {dst}')


def get_row_col_counts(fc):
    """Return the row/columns counts of the figure.

    Args:
        fc: The figure count.

    Returns:
        rc: The row count.
        cc: The column count.
    """
    rc = int(np.sqrt(fc))
    while fc % rc != 0:
        rc -= 1
    cc = int(fc/rc)
    return (rc, cc)
