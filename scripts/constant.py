"""The constant for the simulation script.
"""

# The timescale of the 'ns after start' is ns. Use sec as the unit time.
ONE_SEC = 1000_000_000

# Define the list of styles
CLR_LIST = ['k', 'b', 'g', 'r']  # list of basic colors
STY_LIST = ['-', '--', '-.', ':']  # list of basic linestyles

# Define the target to parse
TARGET = "Confirmation Time (ns)"
ISSUED_MESSAGE = "# of Issued Messages"

# Rename the parameters
VAR_DICT = {'TipsCount': 'k', 'ZipfParameter': 's',
            'NodesCount': 'N', 'MinDelay': 'D'}

# Items for double spending figures
COLORED_CONFIRMED_LIKE_ITEMS = [
    'Blue (Confirmed)', 'Red (Confirmed)', 'Blue (Like)', 'Red (Like)', 'Unconfirmed Blue', 'Unconfirmed Red']

# The color list for the double spending figures
DS_CLR_LIST = ['b', 'r', 'b', 'r', 'b', 'r']
DS_STY_LIST = ['-', '-', '--', '--', "-.", "-."]

# The simulation mapping
SIMULATION_VAR_DICT = {'N': 'nodesCount',
                       'S': 'zipfParameter', 'K': 'tipsCount'}

# The figure naming mapping
FIGURE_NAMING_DICT = {'N': ("NodesCount", "Confirmation Time v.s. Different Node Counts",
                            "Convergence Time v.s. Different Node Counts", "Flips v.s. Different Node Counts",
                            "Unconfirming Counts v.s. Different Node Counts"),
                      'S': ("ZipfParameter", "Confirmation Time v.s. Different Zipf's Parameters",
                            "Convergence Time v.s. Different Zipf's Parameters", "Flips v.s. Different Zipf's Parameters",
                            "Unconfirming Counts v.s. Different Zipf's Parameters"),
                      'K': ("TipsCount", "Confirmation Time v.s. Different Parents Counts",
                            "Convergence Time v.s. Different Parents Counts", "Flips v.s. Different Parents Counts",
                            "Unconfirming Counts v.s. Different Parents Counts"),
                      'D': ("MinDelay", "Confirmation Time v.s. Different Delays",
                            "Convergence Time v.s. Different Delays", "Flips v.s. Different Delays",
                            "Unconfirming Counts v.s. Different Delays")}
