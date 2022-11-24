import matplotlib.pyplot as plt
import numpy as np


def analyze(filename):
    """Analyze the Time Since Confirmation
    """
    x = []
    with open(filename, 'r') as f:
        x = [float(lines.split()[1])
             for lines in f.readlines() if lines[:21] == 'timeSinceConfirmation']
    print(x)
    plt.hist(x, weights=np.ones(len(x))/len(x), bins=10, label="Data")
    plt.ylabel("Probability")
    plt.xlabel("Time Since Confirmation")
    plt.title("Time Since Confirmation Statistics")
    plt.show()
    with open('sc_fishing.txt', 'w') as f:
        f.write('timeSinceConfirmation\n')
        f.writelines([f'{age}\n' for age in x])


# go run . > debug.txt && python3 tip_analysis.py
if __name__ == "__main__":
    analyze('debug.txt')
