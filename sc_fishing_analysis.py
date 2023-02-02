import matplotlib.pyplot as plt
import numpy as np
plt.style.use('plot_style.txt')


def analyze(filename, tag):
    """Analyze the Time Since Confirmation
    """
    x = []
    with open(filename, 'r') as f:
        x = [float(lines.split()[1])
             for lines in f.readlines() if lines.split()[0] == tag]
    print(x)
    plt.hist(x, weights=np.ones(len(x))/len(x),
             bins=10, label="Data")
    plt.xlim(0, 3)
    plt.ylim(0, 0.5)
    plt.ylabel("Probability")
    plt.xlabel(f'{tag} (s)')
    # plt.title(f'{tag} of Every Tip')
    plt.savefig(f'fishing_{tag}.png', transparent=False)
    plt.close()
    # with open('summary_fishing.txt', 'w') as f:
    #     f.write(f'Statistics of {tag} of Every Tip\n')
    #     f.writelines([f'{age}\n' for age in x])


# go run . > debug.txt && python3 tip_analysis.py
if __name__ == "__main__":
    for tag in ['UnconfirmationAge', 'ConfirmationAge', 'UnconfirmationAgeSinceTip', 'ConfirmationAgeSinceTip']:
        analyze('fishing.txt', tag)
