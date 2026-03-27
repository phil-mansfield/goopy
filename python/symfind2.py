
_registry = { }

def register(trigger=None, **kwargs):
    # I'm going to over-comment this because Python decorators are very
    # confusing to me. (I know this will bring shame to my functional
    # programming professors.)

    # (This code runs when the underlying function is declared.)
    
    def decorator(func):
        # (This code runs when the underlying function is declared. func is
        # the decorated function. The difference between this and the previous
        # level is that you actually know what the function is.)

        if trigger in _registry:
            _registry[trigger].append((func, kwargs))
        else:
            _registry[trigger] = [(func, kwargs)]
        
        def wrapped_func(*args, **kwargs):
            # (This code runs when func is called. *args and **kwargs are the
            # same args and kwargs that get passed to func).
            
            # Okay, after all this nonsense, call the actual function.
            func(*args, **kwargs)
            
            return "Oh no, I've stolen your result"
        
        return wrapped_func
    
    return decorator

def plugin():
    print(_registry)

@register(trigger="FleaComb")
def func(x):
    print("Luna says", x)

if __name__ == "__main__":
    plugin()
