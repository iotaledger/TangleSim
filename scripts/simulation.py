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

# The output folder suffix (e.g., ct for confirmation time and ds for double spending)
OUTPUT_FOLDER_SUFFIX = 'ds'

# Transparent figure
TRANSPARENT = False

# The begining x_axis in ns
X_AXIS_BEGIN = 20000_000_000

# The timescale of the 'ns after start' is ns. Use sec as the unit time.
ONE_SEC = 1000_000_000

# The issuance time of colored message (in ns)
COLORED_MSG_ISSUANCE_TIME = 2000_000_000

# Flags of operations
RUN_SIM = True
PLOT_FIGURES = True

# Define the list of styles
clr_list = ['k', 'b', 'g', 'r']  # list of basic colors
sty_list = ['-', '--', '-.', ':']  # list of basic linestyles

# Define the target to parse
target = "Confirmation Time (ns)"
issued_message = "# of Issued Messages"

# Items for double spending figures
colored_confirmed_like_items = [
    'Blue (Confirmed)', 'Red (Confirmed)', 'Blue (Like)', 'Red (Like)']
# The color list for the double spending figures
ds_clr_list = ['b', 'r', 'b', 'r']
ds_sty_list = ['-', '-', '--', '--']


def parse_aw_file(fn, variation):
    """Parse the accumulated weight files.
    """
    logging.info(f'Parsing {fn}...')
    data = pd.read_csv(fn)

    # Chop data before the begining time
    data = data[data['ns since start'] >= X_AXIS_BEGIN]

    # Reset the index to only consider the confirmed msgs from X_AXIS_BEGIN
    data = data.reset_index()

    # Get the configuration setup of this simulation
    # Note currently we only consider the first node
    config_fn = re.sub('aw0', 'aw', fn)
    config_fn = config_fn.replace('.csv', '.config')

    # Opening JSON file
    with open(config_fn) as f:
        c = json.load(f)

    v = c[variation]

    # ns is the time scale of the aw outputs
    x_axis_adjust = float(ONE_SEC) / float(c["DecelerationFactor"])
    return v, data[target], x_axis_adjust


def parse_throughput_file(fn, variation):
    """Parse the throughput files.
    """
    logging.info(f'Parsing {fn}...')
    data = pd.read_csv(fn)

    # Chop data before the begining time
    data = data[data['ns since start'] >= X_AXIS_BEGIN]

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

    # Return the scaled x axis
    x_axis = (data['ns since start'] / float(ONE_SEC) /
              float(c["DecelerationFactor"]))
    return v, (tip_pool_size, processed_messages, issued_messages, x_axis)


def parse_confirmed_color_file(fn, var):
    """Parse the confirmed color files.
    """
    logging.info(f'Parsing {fn}...')
    data = pd.read_csv(fn)

    # Chop data before the begining time
    data = data[data['ns since start'] >= X_AXIS_BEGIN]

    # Get the configuration setup of this simulation
    config_fn = re.sub('cc', 'aw', fn)
    config_fn = config_fn.replace('.csv', '.config')

    # Get the throughput details
    colored_node_counts = data[colored_confirmed_like_items]
    confirmed_time = data['ns since issuance'].iloc[-1]
    confirmed_time /= ONE_SEC

    # Opening JSON file
    with open(config_fn) as f:
        c = json.load(f)

    confirmed_time /= float(c["DecelerationFactor"])

    v = c[var]

    # Return the scaled x axis
    x_axis = (data['ns since start'] / float(ONE_SEC) /
              float(c["DecelerationFactor"]))

    return v, (colored_node_counts, confirmed_time, x_axis)


def move_results(src, dst):
    """Move the files from the source folder to the destination folder.
    """
    if not os.path.isdir(folder):
        os.mkdir(folder)
    os.system(f'mv {src}/*.config {dst}')
    os.system(f'mv {src}/*.csv {dst}')


def confirmed_like_color_plot(var, fs, ofn, fc):
    """Generate the confirmed/like color figure.
    """
    # Init the matplotlib config
    font = {'family': 'Times New Roman',
            'weight': 'bold',
            'size': 8}
    matplotlib.rc('font', **font)

    variation_data = {}
    for f in glob.glob(fs):
        # Confirmation time and the cc data
        v, ct_cc = parse_confirmed_color_file(f, var)
        variation_data[v] = ct_cc

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

    for i, (v, d) in enumerate(sorted(variation_data.items())):
        (nodes, ct, x_axis) = d
        r_loc = i // cn
        c_loc = i % cn

        for j, n in enumerate(nodes.columns):
            axs[r_loc, c_loc].plot(x_axis, nodes[n], label=n,
                                   color=ds_clr_list[j], ls=ds_sty_list[j], linewidth=1)

        # Only put the legend on the first figures
        if i == 0:
            axs[r_loc, c_loc].legend()
        axs[r_loc, c_loc].set(
            xlabel='Time (s)', ylabel='Node Count', title=f'{var} = {v}, {ct:.1f}(s)')

    plt.savefig(f'{FIGURE_OUTPUT_PATH}/{ofn}', transparent=TRANSPARENT)
    plt.close()


def throughput_plot(var, fs, ofn, fc):
    """Generate the throughput figure.
    """
    # Init the matplotlib config
    font = {'family': 'Times New Roman',
            'weight': 'bold',
            'size': 8}
    matplotlib.rc('font', **font)

    variation_data = {}
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
        (tips, processed, issued, x_axis) = tp
        r_loc = i // cn
        c_loc = i % cn

        axs[r_loc, c_loc].plot(x_axis, tips, label='Tip Pool Size',
                               color=clr_list[0], ls=sty_list[0], linewidth=1)
        axs[r_loc, c_loc].plot(x_axis, processed, label='Processed Messages',
                               color=clr_list[1], ls=sty_list[1], linewidth=1)
        axs[r_loc, c_loc].plot(x_axis, issued, label='Issued Messages',
                               color=clr_list[2], ls=sty_list[2], linewidth=1)

        # Only put the legend on the first figures
        if i == 0:
            axs[r_loc, c_loc].legend()
        axs[r_loc, c_loc].set(
            xlabel='Time (s)', ylabel='Message Count', yscale='log', title=f'{var} = {v}')

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
        v, data, x_axis_adjust = parse_aw_file(f, var)
        variation_data[v] = (data, x_axis_adjust)
    for i, (v, d) in enumerate(sorted(variation_data.items())):
        clr = clr_list[i // 4]
        sty = sty_list[i % 4]

        ct_series = np.array(sorted(d[0].values))
        confirmed_msg_counts = np.array(list(d[0].index))
        plt.plot(ct_series / d[1], confirmed_msg_counts / len(confirmed_msg_counts),
                 label=f'{label} = {v}', color=clr, ls=sty)

    plt.xlabel('Confirmation Time (s)')
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
    folder = f'{RESULTS_PATH}/var_nodes_{OUTPUT_FOLDER_SUFFIX}'
    if RUN_SIM:
        deceleration_factors = [2, 3, 3, 4, 10, 15, 15, 20, 25, 30]
        for idx, n in enumerate(range(100, 1001, 100)):
            os.chdir(MULTIVERSE_PATH)
            os.system(
                f'./multiverse_sim --simulationTarget=CT --nodesCount={n} --decelerationFactor={deceleration_factors[idx]}')

        move_results(RESULTS_PATH, folder)

    # Plot the figures
    if PLOT_FIGURES:
        confirmation_time_plot('NodesCount', folder + '/aw*csv',
                               'CT_nodes.png', 'Confirmation Time v.s. Different Node Counts', 'N')

        throughput_plot('NodesCount', folder + '/tp*csv',
                        'CT_nodes_tp.png', 10)

        confirmed_like_color_plot('NodesCount', folder + '/cc*csv',
                                  'DS_nodes_cc.png', 10)

    # Run the simulation for different zipf's distribution
    folder = f'{RESULTS_PATH}/var_zipf_{OUTPUT_FOLDER_SUFFIX}'
    if RUN_SIM:
        for z in range(0, 23, 2):
            par = float(z) / 10.0
            os.chdir(MULTIVERSE_PATH)
            os.system(f'./multiverse_sim --simulationTarget=CT --zipfParameter={par}')

        move_results(RESULTS_PATH, folder)

    # Plot the figures
    if PLOT_FIGURES:
        confirmation_time_plot('ZipfParameter', folder + '/aw*csv', 'CT_zipfs.png',
                               'Confirmation Time v.s. Different Zip\'s Parameters', 's')

        throughput_plot('ZipfParameter', folder + '/tp*csv',
                        'CT_zipfs_tp.png', 12)

        confirmed_like_color_plot('ZipfParameter', folder + '/cc*csv',
                                  'DS_zipfs_cc.png', 12)

    # Run the simulation for different parents counts and Zipf's par
    z_list = ['0.4', '0.7', '0.9', '2.0']
    if RUN_SIM:
        for z in z_list:
            par = float(z)
            for p in range(2, 18, 1):
                os.chdir(
                    MULTIVERSE_PATH)
                os.system(f'./multiverse_sim --simulationTarget=CT --tipsCount={p} --zipfParameter={par}')

            folder = f'{RESULTS_PATH}/var_parents_{OUTPUT_FOLDER_SUFFIX}_z_{z}'
            move_results(RESULTS_PATH, folder)

    # Plot the figures
    if PLOT_FIGURES:
        for z in z_list:
            folder = f'{RESULTS_PATH}/var_parents_{OUTPUT_FOLDER_SUFFIX}_z_{z}'
            confirmation_time_plot('TipsCount', folder + '/aw*csv',
                                   f'CT_parents_z_{z}.png', 'Confirmation Time v.s. Different Parents Counts', 'k')

            throughput_plot('TipsCount', folder + '/tp*csv',
                            f'CT_parents_z_{z}_tp.png', 16)

            confirmed_like_color_plot('TipsCount', folder + '/cc*csv',
                                      f'DS_parents_z_{z}_cc.png', 16)
