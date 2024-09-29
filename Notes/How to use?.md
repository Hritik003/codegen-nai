## How to use DSPy ?

``` python
from dspy import (

signature,
OpenAI,
Predict,
settings

)
```

Never access the `LM` directly like, 

```python 
turbo=('Hi, I am using DSPY...')
```

instead, try to make use of the modules such as 
```python
pred_mod = Predict('question -> answer')
model_out = pred_mod(question='How many moons does jupiter have?')
```

we have created a pre_mod object and used as a `predict module`, and hence I can use the inference using the question. Remember it won't work out if you don't specify `question` as  a argument.

(say), we inspect the query that the model ran just now using 
```python
turbo.inspect(n=3)
```

```
Given the fields `question`, produce the fields `answer`. 

--- Follow the following format. 

Question: ${question} Answer: ${answer} --- Question: How many moons does Jupiter have? Answer: Answer: Jupiter has 79 known moons.
```

But We never instructed the model in the first place, then what was the reason? 

the `Predict` Module was the advisor to the prompt instructing to do this and that!