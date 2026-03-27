import symfind2

@symfind2.register(trigger="HaloProperty")
def vmax(sym):
    part = sym["Part"] # particle data for this halo
    halo = sym["Halo"] # properties of this halo

    # This is all normal vmax/rmax logic
    part = part[part["bound"]]
    r = np.sqrt(np.sum((part["x"] - halo["x"])**2, axis=1))
    r = np.sort(r)
    
    m = sym.mp * (np.arange(len(r)) + 1)
    V = np.sqrt(sym.G * m / r)

    # This block deals with edge cases
    ok = r > sym.eps
    if np.sum(ok) > 0:
        i_max = np.argmax(V[ok]) + np.sum(~ok)
        rmax, vmax = r[i_max], v[i_max]
    elif len(ok) > 0:
        i_max = len(ok) - 1
        rmax, vmax = r[i_max], v[i_max]
    else:
        rmax, vmax = 0.0, 0.0

    # Assignment
    sym["Halo/RMax"], sym["Halo/VMax"] = rmax, vmax

if __name__ == "__main__": symfind2.plugin()
