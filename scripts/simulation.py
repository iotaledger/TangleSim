import glob
import json
import logging
import os
import re

import matplotlib
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

# Data paths
MULTIVERSE_PATH = '/home/piotrek/Documents/iota/multiverse-simulation'
RESULTS_PATH = MULTIVERSE_PATH + "/results"

FIGURE_OUTPUT_PATH = MULTIVERSE_PATH + '/scripts/figures'

# Simulation time
SIMULATION_TIME = 60.0

# Transparent figure
TRANSPARENT = False

# Define the list of styles
clr_list = ['k', 'b', 'g', 'r']  # list of basic colors
sty_list = ['-', '--', '-.', ':']  # list of basic linestyles

# Define the target to parse
target = "Confirmation Time (ns)"
issued_message = "# of Issued Messages"


def parse_aw_file(fn, variation):
    """Parse the accumulated weight files.
    """
    logging.info(f'Parsing {fn}...')
    data = pd.read_csv(fn)

    # Get the configuration setup of this simulation
    # Note currently we only consider the first node
    config_fn = re.sub('aw0', 'aw', fn)
    config_fn = config_fn.replace('.csv', '.config')

    # Opening JSON file
    with open(config_fn) as f:
        c = json.load(f)

    v = c[variation]
    issued_messages = max(data[issued_message])
    return v, data[target], issued_messages


def parse_throughput_file(fn, variation):
    """Parse the throughput files.
    """
    logging.info(f'Parsing {fn}...')
    data = pd.read_csv(fn)

    # Get the configuration setup of this simulation
    config_fn = re.sub('tp', 'aw', fn)
    config_fn = config_fn.replace('.csv', '.config')

    # Get the throughput details
    tip_pool_size = data['UndefinedColor (Tip Pool Size)']
    processed_messages = data['UndefinedColor (Processed)']
    issued_messages = data['# of Issued Messages']

    # Opening JSON file
    with open(config_fn) as f:
        c = json.load(f)

    v = c[variation]

    return v, (tip_pool_size, processed_messages, issued_messages)


def move_results(src, dst):
    """Move the files from the source folder to the destination folder.
    """
    if not os.path.isdir(folder):
        os.mkdir(folder)
    os.system(f'mv {src}/*.config {dst}')
    os.system(f'mv {src}/*.csv {dst}')


def throughput_plot(var, fs, ofn, fc, chop=0):
    """Generate the throughput figure with chopping the first `chop` data points.
    """
    # Init the matplotlib config
    font = {'family': 'Times New Roman',
            'weight': 'bold',
            'size': 8}
    matplotlib.rc('font', **font)

    variation_data = {}
    throughput = []
    for f in glob.glob(fs):
        v, tp = parse_throughput_file(f, var)
        variation_data[v] = tp

    rn = 4
    cn = 4
    if fc == 10:
        rn = 2
        cn = 5
    elif fc <= 12:
        rn = 3
        cn = 4
    fig, axs = plt.subplots(rn, cn, figsize=(
        12, 5), dpi=500, constrained_layout=True)

    for i, (v, tp) in enumerate(sorted(variation_data.items())):
        (tips, processed, issued) = tp
        r_loc = i // cn
        c_loc = i % cn

        axs[r_loc, c_loc].plot(np.array(list(tips.index))[chop:], tips[chop:],
                               label='Tip Pool Size', color=clr_list[0], ls=sty_list[0], linewidth=1)
        axs[r_loc, c_loc].plot(np.array(list(processed.index))[chop:], processed[chop:],
                               label='Processed Messages', color=clr_list[1], ls=sty_list[1], linewidth=1)
        axs[r_loc, c_loc].plot(np.array(list(issued.index))[chop:], issued[chop:],
                               label='Issued Messages', color=clr_list[2], ls=sty_list[2], linewidth=1)
        axs[r_loc, c_loc].legend()
        axs[r_loc, c_loc].set(
            xlabel='Time (100ms)', ylabel='Message Count', yscale='log', title=f'{var} = {v}')

    plt.savefig(f'{FIGURE_OUTPUT_PATH}/{ofn}', transparent=TRANSPARENT)
    plt.close()


def confirmation_time_plot(var, fs, ofn, title, label):
    """Generate the confirmation time figures and the corresponding MPS figures.
    """
    # Init the matplotlib config
    font = {'family': 'Times New Roman',
            'weight': 'bold',
            'size': 14}
    matplotlib.rc('font', **font)

    plt.figure(figsize=(12, 5), dpi=500, constrained_layout=True)
    variation_data = {}
    for f in glob.glob(fs):
        v, data, issued_messages = parse_aw_file(f, var)
        variation_data[v] = (data, issued_messages)
    for i, (v, d) in enumerate(sorted(variation_data.items())):
        clr = clr_list[i // 4]
        sty = sty_list[i % 4]
        plt.plot(sorted(d[0].values), np.array(list(d[0].index)) / len(list(d[0].index)),
                 label=f'{label} = {v}', color=clr, ls=sty)

    plt.xlabel('Confirmation Time (ns)')
    plt.ylabel('Cumulative Confirmed Message Percentage')
    plt.legend()
    plt.title(title)
    plt.savefig(f'{FIGURE_OUTPUT_PATH}/{ofn}', transparent=TRANSPARENT)
    plt.close()


if __name__ == '__main__':
    debug_level = "INFO"
    logging.basicConfig(
        level=debug_level,
        format="%(asctime)s,%(msecs)d %(levelname)s: %(message)s",
        datefmt="%Y%M%D %H:%M:%S",
    )

    # Create the figure output path
    os.makedirs(RESULTS_PATH, exist_ok=True)
    os.makedirs(FIGURE_OUTPUT_PATH, exist_ok=True)

    # Run the simulation for different node counts
    for n in range(100, 1001, 100):
        os.chdir(MULTIVERSE_PATH)
        os.system(f'go run . --nodesCount={n}')
    folder = f'{RESULTS_PATH}/var_nodes_ct'
    move_results(RESULTS_PATH, folder)

    # Plot the figures
    confirmation_time_plot('NodesCount', folder + '/aw*csv',
                           'CT_nodes.png', 'Confirmation Time v.s. Different Node Counts', 'N')

    throughput_plot('NodesCount', folder + '/tp*csv',
                    'CT_nodes_tp.png', 10, chop=0)

    # Run the simulation for different zipf's distribution
    for z in range(0, 21, 2):
        par = float(z) / 10.0
        os.chdir(MULTIVERSE_PATH)
        os.system(f'go run . --zipfParameter={par}')
    folder = f'{RESULTS_PATH}/var_zipf_ct'
    move_results(RESULTS_PATH, folder)

    # Plot the figures
    confirmation_time_plot('ZipfParameter', folder + '/aw*csv', 'CT_zipfs.png',
                           'Confirmation Time v.s. Different Zip\'s Parameters', 's')

    throughput_plot('ZipfParameter', folder + '/tp*csv',
                    'CT_zipfs_tp.png', 11, chop=0)

    # Run the simulation for different parents counts and Zipf's par
    z_list = ['0.4', '0.7', '0.9', '2.0']  # ['0.9']
    for z in z_list:
        par = float(z)
        for p in range(2, 17, 1):
            os.chdir(
                MULTIVERSE_PATH)
            os.system(f'go run . --tipsCount={p} --zipfParameter={par}')

        folder = f'{RESULTS_PATH}/var_parents_ct_z_{z}'
        move_results(RESULTS_PATH, folder)

    # Plot the figures
    os.chdir(RESULTS_PATH)
    for z in z_list:
        folder = f'{RESULTS_PATH}/var_parents_ct_z_{z}'
        confirmation_time_plot('TipsCount', folder + '/aw*csv',
                               f'CT_parents_z_{z}.png', 'Confirmation Time v.s. Different Parents Counts', 'k')

        throughput_plot('TipsCount', folder + '/tp*csv',
                        f'CT_parents_z_{z}_tp.png', 16, chop=0)
