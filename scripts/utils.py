"""The helper functions.
"""

import numpy as np
import matplotlib.pyplot as plt
from matplotlib.lines import Line2D
from networkx.drawing.nx_agraph import graphviz_layout
import matplotlib
import networkx as nx
import pandas as pd
import argparse
import logging
import json
import os
import csv
import matplotlib.colors as mcolors

colors = list(mcolors.TABLEAU_COLORS.values())[:10]
color_dict = {}
for i, color in enumerate(colors):
    color_dict[f'color_{i+1}'] = color + 'CC'
colors = color_dict
colornames = list(colors)
burnPolicyNames = {"ManaBurn": [
    "No Burn", "Anxious", "Greedy (+1)", "Greedy (+10)"], "ICCA+": ["Spammer", " ", "Best Effort"]}


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


def get_diameter(fn, ofn, plot=False, transparent=False):
    """Construct the network graph and return the diameter

    Args:
        fn: The nw- file path.
        ofn: The figure output path.
        plot: Plot the network or not.
        transparent: The generated figure is transparent or not.

    Returns:
        diameter: The network diameter.
    """

    # Init the matplotlib config
    font = {'family': 'Times New Roman',
            'weight': 'bold',
            'size': 14}
    matplotlib.rc('font', **font)

    data = pd.read_csv(fn)

    # Get the network information
    weighted_edges = [tuple(x)
                      for x in data[['Peer ID', 'Neighbor ID', 'Network Delay (ns)']].to_numpy()]
    weighted_edges_pruned = set()
    # Remove the repetitive edges
    for u, v, w in weighted_edges:
        u_v, w = sorted([u, v]), w
        weighted_edges_pruned.add((u_v[0], u_v[1], w))
    weighted_edges = list(weighted_edges_pruned)

    nodes = data.drop_duplicates('Peer ID')['Peer ID'].to_numpy()
    weights = data.drop_duplicates('Peer ID')['Weight'].to_numpy()

    # Construct the graph
    g = nx.Graph()
    g.add_weighted_edges_from(weighted_edges)

    diameter = nx.algorithms.distance_measures.diameter(g)
    if plot == False:
        return diameter

    lengths = {}
    for edge in weighted_edges:
        lengths[(edge[0], edge[1])] = dict(len=edge[2])

    pos = graphviz_layout(g, prog='neato')
    ec = nx.draw_networkx_edges(g, pos, alpha=0.2)
    nc = nx.draw_networkx_nodes(g, pos, nodelist=nodes, node_color=weights,
                                with_labels=False, node_size=10, cmap=plt.cm.jet)

    plt.colorbar(nc).ax.set_ylabel(
        'Weights', rotation=270, fontsize=14, labelpad=14)
    plt.axis('off')

    plt.title(f'{len(nodes)} Nodes, Diameter = {diameter}')
    plt.savefig(ofn, transparent=transparent)
    plt.close()
    return diameter


def parse_per_node_metrics(file):
    with open(file, newline='') as csvfile:
        reader = csv.reader(csvfile, delimiter=',', quotechar='|')
        header = next(reader)
        n_nodes = len(header)-1
        n_data = sum(1 for _ in reader)
        data = np.zeros((n_nodes, n_data))
        times = np.zeros(n_data)
        csvfile.seek(0)
        i = -1
        for row in reader:
            if i < 0:
                i += 1
                continue
            times[i] = int(row[-1])*10**-9
            data[:, i] = row[:-1]
            i += 1
    return data, times


def parse_metric_names(file):
    with open(file, newline='') as csvfile:
        reader = csv.reader(csvfile, delimiter=',', quotechar='|')
        names = []
        for row in reader:
            names.append(row[0])
        return names


def parse_latencies(file, cd):
    with open(file, newline='') as csvfile:
        reader = csv.reader(csvfile, delimiter=',', quotechar='|')
        next(reader)
        latencies = [[] for _ in range(cd['NODES_COUNT'])]
        times = [[] for _ in range(cd['NODES_COUNT'])]
        for row in reader:
            latencies[int(row[0])].append(int(row[2])*10**-9)
            times[int(row[0])].append(int(row[1])*10**-9)
    return latencies


def parse_int_node_attributes(file, cd):
    with open(file, newline='') as csvfile:
        reader = csv.reader(csvfile, delimiter=',', quotechar='|')
        next(reader)
        attributes = np.zeros(cd['NODES_COUNT'], dtype=int)
        for row in reader:
            attributes[int(row[0])] = int(row[1])
    return attributes


def plot_per_node_metric(data, times, cd, title, ylab):
    fig, ax = plt.subplots(figsize=(8, 4))
    ax.grid(linestyle='--')
    ax.set_xlabel("Time (s)")
    ax.set_ylabel(ylab)
    ax.title.set_text(title)
    burnPolicies = cd['BURN_POLICIES']
    weights = cd['WEIGHTS']
    for NodeID in range(cd['NODES_COUNT']):
        ax.plot(times, data[NodeID, :], color=colors[colornames[burnPolicies[NodeID]]],
                linewidth=4*weights[NodeID]/weights[0])
    ax.set_xlim(0, times[-1])
    ax.set_ylim(0)
    bps = list(set(burnPolicies))
    ModeLines = [Line2D([0], [0], color=colors[colornames[bp]], lw=4)
                 for bp in bps]
    fig.legend(ModeLines, [burnPolicyNames[cd["SchedulerType"]][i]
               for i in bps], loc="lower right")
    plt.savefig(f'{cd["SCHEDULER_FIGURE_OUTPUT_PATH"]}/{title}.png', bbox_inches='tight')
    plt.close()


def plot_per_node_wo_spammer_metric(data, times, cd, title, ylab):
    fig, ax = plt.subplots(figsize=(8, 4))
    ax.grid(linestyle='--')
    ax.set_xlabel("Time (s)")
    ax.set_ylabel(ylab)
    ax.title.set_text(title + ' wo Spammer')
    burnPolicies = cd['BURN_POLICIES']
    weights = cd['WEIGHTS']
    for NodeID in range(cd['NODES_COUNT']):
        # Skip spammer
        if burnPolicies[NodeID] == 0:
            continue
        ax.plot(times, data[NodeID, :], color=colors[colornames[burnPolicies[NodeID]]],
                linewidth=4*weights[NodeID]/weights[0])
    ax.set_xlim(0, times[-1])
    ax.set_ylim(0)
    bps = list(set(burnPolicies))
    # ModeLines = [Line2D([0], [0], color=colors[colornames[bp]], lw=4)
    #              for bp in bps]
    # fig.legend(ModeLines, [burnPolicyNames[cd["SchedulerType"]][i]
    #            for i in bps], loc="lower right")
    plt.savefig(f'{cd["SCHEDULER_FIGURE_OUTPUT_PATH"]}/{title}_wo_spammer.png', bbox_inches='tight')
    plt.close()

def plot_per_node_rates(messages, times, cd, title):
    fig, ax = plt.subplots(2, 1, sharex=True, figsize=(8, 8))
    ax[0].grid(linestyle='--')
    ax[0].set_xlabel("Time (s)")
    ax[0].set_ylabel("Rate (Blocks/s)")
    ax[1].grid(linestyle='--')
    ax[1].set_xlabel("Time (s)")
    ax[1].set_ylabel("Scaled Rate")
    ax[0].title.set_text(title)
    avg_window = 50
    burnPolicies = cd['BURN_POLICIES']
    weights = cd['WEIGHTS']
    for NodeID in range(cd['NODES_COUNT']):
        rate = (messages[NodeID, 1:]-messages[NodeID, :-1]) * \
            1000/(cd['MONITOR_INTERVAL'])
        ax[0].plot(times[avg_window:], np.convolve(np.ones(avg_window)/avg_window, rate, 'valid'),
                   color=colors[colornames[burnPolicies[NodeID]]], linewidth=4*weights[NodeID]/weights[0])
        ax[1].plot(times[avg_window:], np.convolve(np.ones(avg_window)/avg_window, rate, 'valid')*sum(weights) /
                   weights[NodeID], color=colors[colornames[burnPolicies[NodeID]]], linewidth=4*weights[NodeID]/weights[0])
    ax[0].set_xlim(0, times[-1])
    ax[1].set_xlim(0, times[-1])
    ax[0].set_ylim(0)
    ax[1].set_ylim(0, 500)
    bps = list(set(burnPolicies))
    ModeLines = [Line2D([0], [0], color=colors[colornames[bp]], lw=4)
                 for bp in bps]
    fig.legend(ModeLines, [burnPolicyNames[cd["SchedulerType"]][i]
               for i in bps], loc="lower right")
    plt.savefig(cd['RESULTS_PATH']+'/'+cd['SCRIPT_START_TIME'] +
                '/figures/'+title+'.png', bbox_inches='tight')
    plt.close()


def plot_latency_cdf(latencies, cd, title):
    fig, ax = plt.subplots(figsize=(8, 4))
    ax.set_xlabel("Latency (s)")
    ax.grid(linestyle='--')
    ax.title.set_text(title)
    maxlats = [max(latencies[NodeID])
               for NodeID in range(len(latencies)) if latencies[NodeID]]
    if not maxlats:
        return
    maxval = max(maxlats)
    nbins = 1000
    bins = np.arange(0, maxval+maxval/nbins, maxval/nbins)
    pdf = np.zeros(len(bins))
    burnPolicies = cd['BURN_POLICIES']
    weights = cd['WEIGHTS']
    for NodeID in range(len(latencies)):
        if not latencies[NodeID]:
            continue
        i = 0
        if latencies[NodeID]:
            lats = sorted(latencies[NodeID])
            for lat in lats:
                while i < len(bins):
                    while lat > bins[i]:
                        i += 1
                    break
                pdf[i-1] += 1
        pdf = pdf/sum(pdf)
        cdf = np.cumsum(pdf)
        ax.plot(bins, cdf, color=colors[colornames[burnPolicies[NodeID]]],
                linewidth=4*weights[NodeID]/weights[0])

    ax.set_xlim(0, bins[-1])
    ax.set_ylim(0, 1.1)
    bps = list(set(burnPolicies))
    ModeLines = [Line2D([0], [0], color=colors[colornames[bp]], lw=4)
                 for bp in bps]
    fig.legend(ModeLines, [burnPolicyNames[cd["SchedulerType"]][i]
               for i in bps], loc="lower right")
    plt.savefig(cd['RESULTS_PATH']+'/'+cd['SCRIPT_START_TIME'] +
                '/figures/'+title+'.png', bbox_inches='tight')
    plt.close()
    
def plot_total_traffic(data, times, cd, title, ylim=None):
    _, ax = plt.subplots(figsize=(8, 4))
    ax.grid(linestyle='--')
    ax.set_xlabel("Time (s)")
    ax.set_ylabel("Rate (Blocks/s)")
    ax.title.set_text(title)
    totals = np.sum(data, axis=0)

    # read config from mb.config
    with open(cd['CONFIGURATION_PATH']) as f:
            c = json.load(f)

    avg_window = 10* (c['SimulationDuration']*1e-9) / 60
    rate = (totals[avg_window:]-totals[:-avg_window]) * \
        100/(avg_window*cd['MONITOR_INTERVAL'])

    plt.bar(times[avg_window:][::avg_window], rate[::avg_window], label='Disseminated Blocks')

    partition = 15* (c['SimulationDuration']*1e-9) / 60
    # remainder = len(times[avg_window:][::10]) % 4
    # print(remainder)
    congestions = ([c['CongestionPeriods'][0]] * (partition + 1) +
                   [c['CongestionPeriods'][1]] * (partition) +
                   [c['CongestionPeriods'][2]] * (partition) +
                   [c['CongestionPeriods'][3]] * (partition))
    plt.plot(congestions, 'r', label='Congestion')
    print(cd["SCHEDULER_FIGURE_OUTPUT_PATH"])

    ax.set_xlim(0, times[-1])
    if ylim is not None:
        ax.set_ylim(0, ylim)
        plt.savefig(f'{cd["SCHEDULER_FIGURE_OUTPUT_PATH"]}/{title}{str(ylim)}.png', bbox_inches='tight')
    else:
        plt.savefig(f'{cd["SCHEDULER_FIGURE_OUTPUT_PATH"]}/{title}.png', bbox_inches='tight')
    plt.close()


def plot_total_rate(data, times, cd, title, ylim=None):
    _, ax = plt.subplots(figsize=(8, 4))
    ax.grid(linestyle='--')
    ax.set_xlabel("Time (s)")
    ax.set_ylabel("Rate (Blocks/s)")
    ax.title.set_text(title)
    totals = np.sum(data, axis=0)
    avg_window = 10
    rate = (totals[avg_window:]-totals[:-avg_window]) * \
        1000/(avg_window*cd['MONITOR_INTERVAL'])
    ax.plot(times[avg_window:], rate, color='k')
    ax.set_xlim(0, times[-1])
    if ylim is not None:
        ax.set_ylim(0, ylim)
        plt.savefig(cd['RESULTS_PATH']+'/'+cd['SCRIPT_START_TIME'] +
                    '/figures/'+title+str(ylim)+'.png', bbox_inches='tight')
    else:
        plt.savefig(cd['RESULTS_PATH']+'/'+cd['SCRIPT_START_TIME'] +
                    '/figures/'+title+'.png', bbox_inches='tight')
    plt.close()

def plot_traffic(data, title, cd):
    plt.clf()
    # Extract data columns
    slot_ids = data['Slot ID']
    blocks_counts = data['Blocks Count']

    # Create a bar plot using Matplotlib
    plt.bar(slot_ids, blocks_counts, label='Stored Blocks')

    # Add axis labels and title
    plt.xlabel('Slot')
    plt.xlim(0, max(slot_ids)+1)
    plt.ylabel('Blocks per Slot')
    plt.title('Traffic')

    # Add a red line representing Congestions
    quarter = len(slot_ids) // 4
    congestions = [50] * quarter + [150] * \
        quarter + [150] * quarter + [50] * quarter
    plt.plot(congestions, 'r', label='Congestion')

    # Add legend
    plt.legend()

    plt.savefig(cd['RESULTS_PATH']+'/'+cd['SCRIPT_START_TIME'] +
                '/figures/'+title+'.png', bbox_inches='tight')
    plt.close()

def plot_latency(latencies, times, cd, title):
    fig, ax = plt.subplots(figsize=(8, 4))
    ax.set_xlabel("Time (s)")
    ax.set_ylabel("Latency (ms)")
    ax.grid(linestyle='--')
    ax.title.set_text(title)
    endtime = max([times[NodeID][-1] for NodeID in range(len(times))])
    nbins = 100
    bins = np.arange(0, endtime, endtime/nbins)
    for NodeID in range(len(latencies)):
        lats = np.zeros(nbins)
        i = 0
        for bin in bins:
            if times[NodeID][i] < bin:
                continue
            # incomplete
