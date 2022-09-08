"""The constant for the simulation script.
"""

# The timescale of the 'ns after start' is ns. Use sec as the unit time.
ONE_SEC = 1000_000_000

# Define the list of styles
CLR_LIST = ['k', 'b', 'g', 'r', 'y', 'purple', 'gray']  # list of basic colors
STY_LIST = ['-', '--', '-.', ':']  # list of basic linestyles

# Define the target to parse
TARGET = "Confirmation Time (ns)"
ISSUED_MESSAGE = "# of Issued Messages"

# Rename the parameters
VAR_DICT = {'ParentsCount': 'k', 'ZipfParameter': 's',
            'NodesCount': 'N', 'MinDelay': 'D', 'ConfirmationThreshold': 'AW', 'AccidentalMana': 'IM', 'AdversaryMana': 'AD',
            'AdversaryNodeCounts': 'AC', 'AdversarySpeedup': 'SU', 'PacketLoss': 'P'}

# Items for double spending figures
COLORED_CONFIRMED_LIKE_ITEMS = [
    'Blue (Confirmed Accumulated Weight)', 'Red (Confirmed Accumulated Weight)', 'Blue (Like Accumulated Weight)',
    'Red (Like Accumulated Weight)']

# The color list for the double spending figures
DS_CLR_LIST = ['b', 'r', 'b', 'r', 'b', 'r']
DS_STY_LIST = ['-', '-', '--', '--', "-.", "-."]

# The simulation mapping
SIMULATION_VAR_DICT = {'N': 'nodesCount',
                       'S': 'zipfParameter',
                       'K': 'parentsCount',
                       'P': 'packetLoss'}

# The figure naming mapping
FIGURE_NAMING_DICT = {'N': ("NodesCount", "Confirmation Time v.s. Different Node Counts",
                            "Convergence Time v.s. Different Node Counts", "Flips v.s. Different Node Counts",
                            "Unconfirming Counts v.s. Different Node Counts",
                            "Confirmation Weight Depth v.s. Different Node Counts"),
                      'S': ("ZipfParameter", "Confirmation Time v.s. Different Zipf's Parameters",
                            "Convergence Time v.s. Different Zipf's Parameters",
                            "Flips v.s. Different Zipf's Parameters",
                            "Unconfirming Counts v.s. Different Zipf's Parameters",
                            "Confirmation Weight Depth v.s. Different Zipf's Parameters"),
                      'K': ("ParentsCount", "Confirmation Time v.s. Different Parents Counts",
                            "Convergence Time v.s. Different Parents Counts", "Flips v.s. Different Parents Counts",
                            "Unconfirming Counts v.s. Different Parents Counts",
                            "Confirmation Weight Depth v.s. Different Parents Counts"),
                      'D': ("MinDelay", "Confirmation Time v.s. Different Delays",
                            "Convergence Time v.s. Different Delays", "Flips v.s. Different Delays",
                            "Unconfirming Counts v.s. Different Delays",
                            "Confirmation Weight Depth v.s. Different Delays"),
                      'P': ("PacketLoss", "Confirmation Time v.s. Different Packet Losses",
                            "Convergence Time v.s. Different Packet Losses", "Flips v.s. Different Packet Losses",
                            "Unconfirming Counts v.s. Different Packet Losses",
                            "Confirmation Weight Depth v.s. Different Packet Losses"),
                      'AW': ("ConfirmationThreshold", "Confirmation Time v.s. Different Thresholds",
                             "Convergence Time v.s. Different Threshold", "Flips v.s. Different Thresholds",
                             "Unconfirming Counts v.s. Different Thresholds",
                             "Confirmation Weight Depth v.s. Different Thresholds"),
                      'IM': ("AccidentalMana", "Confirmation Time v.s. Different Issuers",
                             "Convergence Time v.s. Different Issuers", "Flips v.s. Different Issuers",
                             "Unconfirming Counts v.s. Different Issuers",
                             "Confirmation Weight Depth v.s. Different Issuers"),
                      'AD': ("AdversaryMana", "Confirmation Time v.s. Different Adversary Weights",
                             "Convergence Time v.s. Different Adversary Weights",
                             "Flips v.s. Different Adversary Weights",
                             "Unconfirming Counts v.s. Different Adversary Weights",
                             "Confirmation Weight Depth v.s. Different Adversary Weights"),
                      'AC': ("AdversaryNodeCounts", "Confirmation Time v.s. Different Adversary Node Counts",
                             "Convergence Time v.s. Different Adversary Node Counts",
                             "Flips v.s. Different Adversary Node Counts",
                             "Unconfirming Counts v.s. Different Adversary Node Counts",
                             "Confirmation Weight Depth v.s. Different Adversary Node Counts"),
                      'BS': ("AdversaryMana", "Confirmation Time v.s. Different Adversary Weights",
                             "Convergence Time v.s. Different Adversary Weights",
                             "Flips v.s. Different Adversary Weights",
                             "Unconfirming Counts v.s. Different Adversary Weights",
                             "Confirmation Weight Depth v.s. Different Adversary Weights"),
                      'SU': ("AdversarySpeedup", "Confirmation Time v.s. Different Adversary Speed",
                             "Convergence Time v.s. Different Adversary Speed",
                             "Flips v.s. Different Adversary Speed",
                             "Unconfirming Counts v.s. Different Adversary Speed",
                             "Confirmation Weight Depth v.s. Different Adversary Speed")}
