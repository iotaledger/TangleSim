"""The plotting module to plot the figures.
"""
import glob

import matplotlib
import matplotlib.pyplot as plt
import matplotlib.ticker as mtick
import logging

import constant as c
from parsing import FileParser
from utils import *

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

        variation_data = {}
        for f in glob.glob(fs):
            try:
                v, tp = self.parser.parse_all_throughput_file(f, var)
            except:
                logging.error(f'{fs}: Incomplete Data!')
                continue
            variation_data[v] = tp

        rc, cc = get_row_col_counts(fc)
        fig, axs = plt.subplots(rc, cc, figsize=(
            12, 5), dpi=500, constrained_layout=True)

        for i, (v, tp) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            (tips, x_axis) = tp
            r_loc = i // cc
            c_loc = i % cc

            if fc == 1:
                ax = axs
            elif rc == 1:
                ax = axs[c_loc]
            else:
                ax = axs[r_loc, c_loc]

            max_node_to_plot = 5
            for t in tips:
                max_node_to_plot -= 1
                if max_node_to_plot < 0:
                    break
                ax.plot(x_axis, tips[t], label=t, linewidth=1)

            # Only put the legend on the first figures
            if i == 0:
                ax.legend()
            ax.set(
                xlabel='Time (s)', ylabel='Block Count', yscale='log', title=f'{self.var_dict[var]} = {v}')

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
        for i, (v, d) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]))):
            z = (d[0]*1e-9).tolist()
            data.append((d[0]*1e-9).tolist())
            variations.append(v)

        plt.violinplot(data)
        plt.xlabel(label)
        # plt.xlabel('Zipf Parameter')
        plt.xticks(ticks=list(range(1, 1 + len(variations))),
                   labels=variations)

        # axes = plt.axes()
        # axes.set_ylim([0, 11])
        plt.ylabel('Confirmation Time (s)')
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

        plt.figure(figsize=(6, 6), dpi=500, constrained_layout=True)

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
        colored_v = set()
        for i, (v, d_list) in enumerate(sorted(variation_data.items(), key=lambda item: eval(item[0]), reverse=True)):
            for l in d_list:
                (ww, x_axis) = l
                clr = self.clr_list[i % 7]
                sty = self.sty_list[0]
                if v in colored_v:
                    plt.plot(x_axis, 100.0 * ww, color=clr, ls=sty)
                else:
                    plt.plot(x_axis, 100.0 * ww,
                             label=f'{label} = {v}', color=clr, ls=sty)
                    colored_v.add(v)

        plt.gca().yaxis.set_major_formatter(mtick.PercentFormatter())
        plt.xlabel('Time (s)')
        # plt.xlim(0, 5)
        plt.ylabel('Witness Weight (%)')
        plt.legend()
        plt.title('Witness Weight v.s. Time')
        plt.savefig(f'{self.figure_output_path}/{ofn}',
                    transparent=self.transparent)
        plt.close()
