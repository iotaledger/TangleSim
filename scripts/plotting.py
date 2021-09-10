"""The plotting module to plot the figures.
"""
import glob
import matplotlib
import matplotlib.pyplot as plt
import json
import logging
import re
import numpy as np

import constant as c
from utils import *
from parsing import FileParser


class FigurePlotter:
    """
    The figure plotter of the simulation results.
    """

    def __init__(self, cd):
        """Initialize the parser and the constant/configuration values.
        """
        self.parser = FileParser(cd)
        self.figure_output_path = cd['FIGURE_OUTPUT_PATH']
        self.transparent = cd['TRANSPARENT']
        self.clr_list = c.CLR_LIST
        self.sty_list = c.STY_LIST
        self.ds_clr_list = c.DS_CLR_LIST
        self.ds_sty_list = c.DS_STY_LIST
        self.var_dict = c.VAR_DICT

    def convergence_time_distribution_plot(self, var, base_folder, ofn, fc, iters, title):
        """Generate the convergence time distribution figures.

        Args:
            var: The variation.
            base_folder: The parent folder of the iteration results.
            ofn: The output file name.
            fc: The figure count.
            iters: The number of iteration.
            title: The figure title.
        """
        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 14}
        matplotlib.rc('font', **font)

        plt.figure(figsize=(12, 5), dpi=500, constrained_layout=True)
        variation_data = {}

        for i in range(iters):
            fs = base_folder + f'/iter_{i}/cc*csv'

            for f in glob.glob(fs):

                # Confirmation time and the cc data
                v, ct_cc = self.parser.parse_confirmed_color_file(f, var)

                # Store the confirmation time
                if v not in variation_data:
                    variation_data[v] = [ct_cc[1]]
                else:
                    variation_data[v].append(ct_cc[1])

        data = []
        variations = []
        for i, (v, d) in enumerate(sorted(variation_data.items())):
            data.append(d)
            variations.append(v)

        plt.boxplot(data)
        plt.xlabel(var)
        plt.title(title)
        plt.xticks(ticks=list(range(1, 1+len(variations))), labels=variations)
        plt.ylabel('Convergence Time (s)')
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()

    def confirmed_like_color_plot(self, var, fs, ofn, fc):
        """Generate the confirmed/like color figures.

        Args:
            var: The variation.
            fs: The path of files.
            ofn: The output file name.
            fc: The figure count.
        """
        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 8}
        matplotlib.rc('font', **font)

        variation_data = {}
        for f in glob.glob(fs):
            # Confirmation time and the cc data
            v, ct_cc = self.parser.parse_confirmed_color_file(f, var)
            variation_data[v] = ct_cc

        rn, cn = get_row_col_counts(fc)
        fig, axs = plt.subplots(rn, cn, figsize=(
            12, 5), dpi=500, constrained_layout=True)

        for i, (v, d) in enumerate(sorted(variation_data.items())):
            (nodes, ct, x_axis) = d
            r_loc = i // cn
            c_loc = i % cn

            for j, n in enumerate(nodes.columns):
                axs[r_loc, c_loc].plot(x_axis, nodes[n], label=n,
                                       color=self.ds_clr_list[j], ls=self.ds_sty_list[j], linewidth=1)

            # Only put the legend on the first figures
            if i == 0:
                axs[r_loc, c_loc].legend()
            axs[r_loc, c_loc].set(
                xlabel='Time (s)', ylabel='Node Count', title=f'{self.var_dict[var]} = {v}, {ct:.1f}(s)')

        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()

    def throughput_plot(self, var, fs, ofn, fc):
        """Generate the throughput figures.

        Args:
            var: The variation.
            fs: The path of files.
            ofn: The output file name.
            fc: The figure count.
        """
        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 8}
        matplotlib.rc('font', **font)

        variation_data = {}
        for f in glob.glob(fs):
            v, tp = self.parser.parse_throughput_file(f, var)
            variation_data[v] = tp

        rn, cn = get_row_col_counts(fc)
        fig, axs = plt.subplots(rn, cn, figsize=(
            12, 5), dpi=500, constrained_layout=True)

        for i, (v, tp) in enumerate(sorted(variation_data.items())):
            (tips, processed, issued, x_axis) = tp
            r_loc = i // cn
            c_loc = i % cn

            axs[r_loc, c_loc].plot(x_axis, tips, label='Tip Pool Size',
                                   color=self.clr_list[0], ls=self.sty_list[0], linewidth=1)
            axs[r_loc, c_loc].plot(x_axis, processed, label='Processed Messages',
                                   color=self.clr_list[1], ls=self.sty_list[1], linewidth=1)
            axs[r_loc, c_loc].plot(x_axis, issued, label='Issued Messages',
                                   color=self.clr_list[2], ls=self.sty_list[2], linewidth=1)

            # Only put the legend on the first figures
            if i == 0:
                axs[r_loc, c_loc].legend()
            axs[r_loc, c_loc].set(
                xlabel='Time (s)', ylabel='Message Count', yscale='log', title=f'{self.var_dict[var]} = {v}')

        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()

    def confirmation_time_plot(self, var, fs, ofn, title, label):
        """Generate the confirmation time figures.

        Args:
            var: The variation.
            fs: The path of files.
            ofn: The output file name.
            title: The plot title.
            label: The curve label.
        """
        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 14}
        matplotlib.rc('font', **font)

        plt.figure(figsize=(12, 5), dpi=500, constrained_layout=True)
        variation_data = {}
        for f in glob.glob(fs):
            v, data, x_axis_adjust = self.parser.parse_aw_file(f, var)
            variation_data[v] = (data, x_axis_adjust)
        for i, (v, d) in enumerate(sorted(variation_data.items())):
            clr = self.clr_list[i // 4]
            sty = self.sty_list[i % 4]

            ct_series = np.array(sorted(d[0].values))
            confirmed_msg_counts = np.array(list(d[0].index))
            plt.plot(ct_series / d[1], confirmed_msg_counts / len(confirmed_msg_counts),
                     label=f'{label} = {v}', color=clr, ls=sty)

        plt.xlabel('Confirmation Time (s)')
        plt.ylabel('Cumulative Confirmed Message Percentage')
        plt.legend()
        plt.title(title)
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()
