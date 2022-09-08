"""The plotting module to plot the figures.
"""
import glob

import matplotlib
import matplotlib.pyplot as plt
import matplotlib.ticker as mtick
import scipy.stats as st
import logging

import constant as c
from parsing import FileParser
from utils import *
from collections import defaultdict

plt.style.use('plot_style.txt')


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

    def _distribution_boxplot(self, var, base_folder, ofn, fc, iters, title, target):
        """The basic function of generating the distribution boxplot figures.
        Args:
            var: The variation.
            base_folder: The parent folder of the iteration results.
            ofn: The output file name.
            fc: The figure count.
            iters: The number of iteration.
            title: The figure title.
            target: The variation, should be 'flips', 'convergence_time'.
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

                try:
                    # colored_node_aw, convergence_time, flips, unconfirming blue, unconfirming red, honest total weight, x_axis
                    v, (cc, ct, flips, *_,
                        x) = self.parser.parse_confirmed_color_file(f, var)
                except:
                    logging.error(f'{fs}: Incomplete Data!')
                    continue

                if target == 'convergence_time':
                    # Store the convergence time
                    if v not in variation_data:
                        variation_data[v] = [ct]
                    else:
                        variation_data[v].append(ct)

                elif target == 'flips':
                    # Store the flips
                    if v not in variation_data:
                        variation_data[v] = [flips]
                    else:
                        variation_data[v].append(flips)

        data = []
        variations = []
        for i, (v, d) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            data.append(d)
            variations.append(v)

        plt.boxplot(data)
        plt.xlabel(var)
        plt.title(title)
        plt.xticks(ticks=list(range(1, 1 + len(variations))),
                   labels=variations)
        if target == 'convergence_time':
            plt.ylabel('Convergence Time (s)')
        elif target == 'flips':
            plt.ylabel('Number of Flips')
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()

    def flips_distribution_plot(self, var, base_folder, ofn, fc, iters, title):
        """Generate the flips distribution figures.
        Args:
            var: The variation.
            base_folder: The parent folder of the iteration results.
            ofn: The output file name.
            fc: The figure count.
            iters: The number of iteration.
            title: The figure title.
        """
        self._distribution_boxplot(
            var, base_folder, ofn, fc, iters, title, 'flips')

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
        self._distribution_boxplot(
            var, base_folder, ofn, fc, iters, title, 'convergence_time')

    def unconfirmed_count_distribution_plot(self, var, base_folder, ofn, fc, iters, title):
        """The basic function of generating the distribution boxplot figures.
        Args:
            var: The variation.
            base_folder: The parent folder of the iteration results.
            ofn: The output file name.
            fc: The figure count.
            iters: The number of iteration.
            title: The figure title.
        """

        def set_box_color(bp, color):
            """The helper functions to set colors of the unconfirming boxplot.
            """
            plt.setp(bp['boxes'], color=color)
            plt.setp(bp['whiskers'], color=color)
            plt.setp(bp['caps'], color=color)
            plt.setp(bp['medians'], color=color)

        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 14}
        matplotlib.rc('font', **font)

        plt.figure(figsize=(12, 5), dpi=500, constrained_layout=True)
        variation_data_blue = {}
        variation_data_red = {}

        for i in range(iters):
            fs = base_folder + f'/iter_{i}/cc*csv'

            for f in glob.glob(fs):

                try:
                    # colored_node_counts, convergence_time, flips, unconfirming blue, unconfirming red, honest total weight, x_axis
                    v, (cc, *_, ub, ur, _,
                        x) = self.parser.parse_confirmed_color_file(f, var)
                except:
                    logging.error(f'{fs}: Incomplete Data!')
                    continue

                # Store the unconfirming counts
                if v not in variation_data_blue:
                    variation_data_blue[v] = [ub]
                    variation_data_red[v] = [ur]
                else:
                    variation_data_blue[v].append(ub)
                    variation_data_red[v].append(ur)

        data_blue = []
        data_red = []
        variations = []
        for v, d in sorted(variation_data_blue.items(), key=lambda item: eval(item[0])):
            data_blue.append(d)
            data_red.append(variation_data_red[v])
            variations.append(v)

        # Location of the box
        box_location = 0
        xticks = []
        for i in range(len(variations)):
            box_location += 1
            xticks.append(box_location + 0.5)
            bp = plt.boxplot(data_blue[i], positions=[
                box_location], sym='o', widths=0.6)
            set_box_color(bp, 'b')
            box_location += 1
            bp = plt.boxplot(data_red[i], positions=[
                box_location], sym='x', widths=0.6)
            box_location += 1
            set_box_color(bp, 'r')

        # draw temporary red and blue lines and use them to create a legend
        h_b, = plt.plot([1, 1], 'b-')
        h_r, = plt.plot([1, 1], 'r-')
        plt.legend((h_b, h_r), ('Unconfirming Blue', 'Unconfirming Red'))
        h_b.set_visible(False)
        h_r.set_visible(False)

        plt.xlabel(var)
        plt.ylabel('Counts')
        plt.title(title)
        plt.xticks(xticks, variations)
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()

    def confirmation_depth_distribution_plot(self, var, base_folder, ofn, fc, iters, title):
        """The function of generating the confirmation-depth distribution boxplot figures.
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
            fs = base_folder + f'/iter_{i}/nd*csv'

            for f in glob.glob(fs):

                try:
                    # confirmation_rate_depth
                    v, depth = self.parser.parse_node_file(f, var)
                except:
                    logging.error(f'{fs}: Incomplete Data!')
                    continue

                if v not in variation_data:
                    variation_data[v] = [depth]
                else:
                    variation_data[v].append(depth)

        data = []
        variations = []
        for i, (v, d) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            data.append(d)
            variations.append(v)

        plt.boxplot(data)
        plt.xlabel(var)
        plt.title(title)
        plt.xticks(ticks=list(range(1, 1 + len(variations))),
                   labels=variations)

        plt.ylabel('Confirmation Weight Depth (%)')
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
            try:
                # colored_node_aw, convergence_time, flips, unconfirming blue, unconfirming red, honest total weight, x_axis
                v, cc_ct_flips_total_aw_x = self.parser.parse_confirmed_color_file(
                    f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue

            variation_data[v] = cc_ct_flips_total_aw_x

        rc, cc = get_row_col_counts(fc)
        fig, axs = plt.subplots(rc, cc, figsize=(
            12, 5), dpi=500, constrained_layout=True)

        for i, (v, d) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            (weights, ct, *_, total_aw, x_axis) = d
            r_loc = i // cc
            c_loc = i % cc

            if fc == 1:
                ax = axs
            elif rc == 1:
                ax = axs[c_loc]
            else:
                ax = axs[r_loc, c_loc]
            for j, n in enumerate(weights.columns):
                aw_percentage = 100.0 * weights[n] / total_aw
                ax.plot(x_axis, aw_percentage, label=n,
                        color=self.ds_clr_list[j], ls=self.ds_sty_list[j], linewidth=1)

            # Only put the legend on the first figures
            if i == 0:
                ax.legend()
            ax.yaxis.set_major_formatter(mtick.PercentFormatter())
            ax.set(xlabel='Time (s)', ylabel='Accumulated Weight Percentage',
                   title=f'{self.var_dict[var]} = {v}, {ct:.1f}(s)')

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
            try:
                v, tp = self.parser.parse_throughput_file(f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue
            variation_data[v] = tp

        rc, cc = get_row_col_counts(fc)
        fig, axs = plt.subplots(rc, cc, figsize=(
            12, 5), dpi=500, constrained_layout=True)

        for i, (v, tp) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            (tips, processed, issued, x_axis) = tp
            r_loc = i // cc
            c_loc = i % cc

            if fc == 1:
                ax = axs
            elif rc == 1:
                ax = axs[c_loc]
            else:
                ax = axs[r_loc, c_loc]

            ax.plot(x_axis, tips, label='Tip Pool Size',
                    color=self.clr_list[0], ls=self.sty_list[0], linewidth=1)
            ax.plot(x_axis, processed, label='Processed Messages',
                    color=self.clr_list[1], ls=self.sty_list[1], linewidth=1)
            ax.plot(x_axis, issued, label='Issued Messages',
                    color=self.clr_list[2], ls=self.sty_list[2], linewidth=1)

            # Only put the legend on the first figures
            if i == 0:
                ax.legend()
            ax.set(
                xlabel='Time (s)', ylabel='Message Count', yscale='log', title=f'{self.var_dict[var]} = {v}')

        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()

    def throughput_all_plot(self, var, fs, ofn, fc):
        """Generate the throughput figures of all nodes.
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

        # Colors
        BG_WHITE = "#fbf9f4"
        GREY_LIGHT = "#b4aea9"
        GREY50 = "#7F7F7F"
        BLUE_DARK = "#1B2838"
        BLUE = "#2a475e"
        BLACK = "#282724"
        GREY_DARK = "#747473"
        RED_DARK = "#850e00"

        # Colors taken from Dark2 palette in RColorBrewer R library
        COLOR_SCALE = ["#1B9E77", "#D95F02", "#7570B3", '#2a475e']

        # Horizontal positions for the violins.
        # They are arbitrary numbers. They could have been [-1, 0, 1] for example.
        POSITIONS = [0, 1, 2, 3]

        # Horizontal lines
        HLINES = [40, 50, 60]

        variation_data = {}
        for f in glob.glob(fs):
            try:
                v, tp = self.parser.parse_all_throughput_file(f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue
            variation_data[v] = tp

        cc, rc = get_row_col_counts(fc)
        fig, axs = plt.subplots(rc, cc, constrained_layout=False)

        for i, (v, tp) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            (tips, x_axis) = tp
            r_loc = i // cc
            c_loc = i % cc

            if fc == 1:
                ax = axs
            elif cc == 1:
                ax = axs[r_loc]
            else:
                ax = axs[r_loc, c_loc]

            nodes = ['Node 0', 'Node 1', 'Node 98', 'Node 99']

            # Only put the legend on the first figures
            if i == 0:
                ax.legend(ncol=4)
            ax.set(
                xlabel='Node ID', ylabel='Tip Pool Size')
            ax.set_xticklabels(nodes)
            # ax.set_title(
            #     f'Uniform Random Delay = {int(v)}–{int(v)+100} (ms)', fontsize=12)
            # if int(v) == 450:
            #     ax.set_title(
            #         f'Uniform Random Delay = 450–550 (ms)', fontsize=12)
            # else:
            #     ax.set_title(
            #         f'Uniform Random Delay = 50–950 (ms)', fontsize=12)
            # ax.set_title(
            #     f'Exponential Delay Distribution with Mean = {int(v)} (ms)', fontsize=12)
            # if int(v) == 100:
            #     ax.set_title(
            #         f'Node Count = 100, TPS = 100', fontsize=12)
            # else:
            #     ax.set_title(
            #         f'Node Count = 500, TPS = 4000', fontsize=12)
            #     ax.set_ylim((0, 3000))
            # ax.set_title(
            #     f'Constant Network Delay = {int(v)} (ms)', fontsize=12)
            ax.set_title(
                f'Node Count = {int(v)}', fontsize=12)
            ax.set_ylim((0, 60))
            ax.set_yticks(range(0, 61, 30))
            # ax.set_xlim((20, 60))

            y_data = (tips[nodes[0]].values,
                      tips[nodes[1]].values,
                      tips[nodes[2]].values,
                      tips[nodes[3]].values)
            # Some layout stuff ----------------------------------------------
            # Background color
            # fig.patch.set_facecolor(BG_WHITE)
            # ax.set_facecolor(BG_WHITE)

            # Horizontal lines that are used as scale reference
            # for h in HLINES:
            #     ax.axhline(h, color=GREY50, ls=(0, (5, 5)), alpha=0.8, zorder=0)

            # Add violins ----------------------------------------------------
            # bw_method="silverman" means the bandwidth of the kernel density
            # estimator is computed via Silverman's rule of thumb.
            # More on this in the bonus track ;)

            # The output is stored in 'violins', used to customize their appearence
            violins = ax.violinplot(
                y_data,
                positions=POSITIONS,
                widths=0.45,
                bw_method="silverman",
                showmeans=False,
                showmedians=False,
                showextrema=False
            )

            # Customize violins (remove fill, customize line, etc.)
            for pc in violins["bodies"]:
                pc.set_facecolor("none")
                pc.set_edgecolor(BLACK)
                pc.set_linewidth(0.4)
                pc.set_alpha(1)

            # Add boxplots ---------------------------------------------------
            # Note that properties about the median and the box are passed
            # as dictionaries.

            medianprops = dict(
                linewidth=2,
                color=GREY_DARK,
                solid_capstyle="butt"
            )
            boxprops = dict(
                linewidth=0.4,
                color=GREY_DARK
            )

            ax.boxplot(
                y_data,
                positions=POSITIONS,
                showfliers=False,  # Do not show the outliers beyond the caps.
                showcaps=False,   # Do not show the caps
                medianprops=medianprops,
                whiskerprops=boxprops,
                boxprops=boxprops
            )

            jitter = 0.04
            x_data = [np.array([i] * len(d)) for i, d in enumerate(y_data)]
            x_jittered = [x + st.t(df=6, scale=jitter).rvs(len(x))
                          for x in x_data]
            # Add jittered dots ----------------------------------------------
            for x, y, color in zip(x_jittered, y_data, COLOR_SCALE):
                ax.scatter(x, y, s=8, color=color, alpha=0.4)

            # Add mean value labels ------------------------------------------
            means = [y.mean() for y in y_data]
            for i, mean in enumerate(means):
                # Add dot representing the mean
                ax.scatter(i, mean, s=50, color=RED_DARK, zorder=3)

                # Add line conecting mean value and its label
                ax.plot([i, i + 0.25], [mean, mean],
                        ls="dashdot", color="black", zorder=3)

                # Add mean value label.
                ax.text(
                    i + 0.25,
                    mean,
                    r"$\hat{\mu}_{\rm{mean}} = $" + str(round(mean, 2)),
                    fontsize=10,
                    va="center",
                    bbox=dict(
                        facecolor="white",
                        edgecolor="black",
                        boxstyle="round",
                        pad=0.15
                    ),
                    zorder=10  # to make sure the line is on top
                )

        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()

    def confirmation_time_violinplot(self, var, fs, ofn, title, label):

        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 14}
        matplotlib.rc('font', **font)

        plt.close('all')
        plt.figure()
        variation_data = {}
        for f in glob.glob(fs):
            try:
                v, data, x_axis_adjust = self.parser.parse_aw_file(f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue
            variation_data[v] = (data, x_axis_adjust)

        data = []
        variations = []

        # thorughput = [100, 4000]
        for i, (v, d) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            z = (d[0]*1e-9).tolist()
            if len(z) == 0:
                continue
            # data.append((d[0]*1e-9).tolist())

            data.append(z)
            # print(z)
            # variations.append(f'[{int(v)}, {int(v)+100}]')
            # variations.append(f'{int(v)}–{int(v)+100}')

            # delay = '50–950'
            # if int(v) == 450:
            #     delay = '450–550'
            # ax.set_title(
            #         f'Uniform Random Delay = 450–550 (ms)', fontsize=12)
            # else:
            #     ax.set_title(
            #         f'Uniform Random Delay = 50–950 (ms)', fontsize=12)
            # variations.append(f'{v}, {thorughput[i]}')
            # variations.append(float(v)*100)
            print(z, v)
            variations.append(int(eval(v)[0]))

        # print(data)
        plt.violinplot(data)
        plt.xlabel('Adversary Weight (%)')
        # plt.xlabel('Uniform Random Network Delay (ms)')
        # plt.xlabel('Node Count, TPS')
        # plt.ylim(0, 6)
        # plt.xlabel('Zipf Parameter')
        plt.xticks(ticks=list(range(1, 1 + len(variations))),
                   labels=variations)

        axes = plt.axes()
        axes.set_ylim([0, 6])
        plt.ylabel('Confirmation Time (s)')
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent, dpi=300)
        plt.close()

    def number_of_requested_missing_messages_batch(self, var, fs, ofn, title, label):
        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 14}
        matplotlib.rc('font', **font)

        plt.close('all')
        plt.figure()
        variation_data = defaultdict(list)
        for k, v in variation_data.items():
            print(k, v)
        for f in glob.glob(fs):
            try:
                v, data = self.parser.parse_mm_file(f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue
            variation_data[float(v)*100].append(data)

        variations = []
        data = []

        for i, (v, d) in enumerate(sorted(variation_data.items())):
            variations.append(v)
            data.append(d)
        plt.violinplot(data)

        plt.xlabel('Payload Loss (%)')
        plt.ylim(0, 3000)
        plt.xticks(ticks=list(range(1, 1 + len(variations))),
                   labels=variations)

        plt.ylabel('Number of Requested Messages')
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent, dpi=300)
        plt.close()

    def number_of_requested_missing_messages(self, var, fs, ofn, title, label):
        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 14}
        matplotlib.rc('font', **font)

        plt.close('all')
        plt.figure()
        variation_data = {}
        for f in glob.glob(fs):
            try:
                v, data = self.parser.parse_mm_file(f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue
            variation_data[v] = data

        data = []
        variations = []

        # thorughput = [100, 4000]
        for i, (v, d) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            if float(v)*100 <= 80:
                data.append(d)
                variations.append(float(v)*100)

        print(data, variations)
        plt.bar(range(1, len(variations)+1), data)
        plt.xlabel('Payload Loss (%)')
        # plt.xlabel('Node Count, TPS')
        plt.ylim(0, 2000)
        # plt.xlabel('Zipf Parameter')
        plt.xticks(ticks=list(range(1, 1 + len(variations))),
                   labels=variations)

        plt.ylabel('Number of Requested Messages')
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent, dpi=300)
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
            try:
                v, data, x_axis_adjust = self.parser.parse_aw_file(f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue
            variation_data[v] = (data, x_axis_adjust)
        for i, (v, d) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            clr = self.clr_list[i // 4]
            sty = self.sty_list[i % 4]

            ct_series = np.array(sorted(d[0].values))
            confirmed_msg_counts = np.array(list(d[0].index))
            plt.plot(ct_series / d[1], 100.0 * confirmed_msg_counts / len(confirmed_msg_counts),
                     label=f'{label} = {v}', color=clr, ls=sty)

        plt.gca().yaxis.set_major_formatter(mtick.PercentFormatter())
        plt.xlabel('Confirmation Time (s)')
        plt.ylabel('Cumulative Confirmed Message Percentage')
        plt.legend()
        plt.title(title)
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()

    def witness_weight_plot(self, var, base_folder, ofn, label, repetition):
        """Generate the witness weight figures.
        Args:
            var: The variation.
            base_folder: The base folder.
            ofn: The output file name.
            label: The curve label.
            repetition: The iteration count
        """
        # Init the matplotlib config
        font = {'family': 'Times New Roman',
                'weight': 'bold',
                'size': 14}
        matplotlib.rc('font', **font)

        plt.close('all')
        plt.figure()

        variation_data = {}
        if repetition != 1:
            fs = base_folder + f'/iter_*/ww*csv'
        else:
            fs = base_folder + '/ww*csv'

        for f in glob.glob(fs):
            try:
                v, data, x_axis_adjust = self.parser.parse_ww_file(f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue
            if v not in variation_data:
                variation_data[v] = [(data, x_axis_adjust)]
            else:
                variation_data[v].append((data, x_axis_adjust))

        for i, (v, d_list) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]), reverse=True)):
            for l in d_list:
                (ww, x_axis) = l
                # plt.plot(x_axis, 100.0 * ww,
                #          label=f'Delay = {int(v)}–{int(v)+100} (ms)')
                if int(v) == 100:
                    plt.plot(x_axis, 100.0 * ww,
                             label=f'Node Count = 100, TPS = 100')
                else:
                    plt.plot(x_axis, 100.0 * ww,
                             label=f'Node Count = 500, TPS = 4000')

        plt.gca().yaxis.set_major_formatter(mtick.PercentFormatter())
        plt.xlabel('Time (s)')
        plt.xticks(range(0, 21, 5))
        plt.xlim(0, 20)
        plt.ylim(0, 100)
        plt.ylabel('Witness Weight (%)')
        plt.legend()
        # plt.title('Witness Weight v.s. Time')
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()
