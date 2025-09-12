# Pruebas para la máquina de bytecode (Go)

    Casos correctos

### 1_print.bytecode
def prueba1():
    print("hola mundo")
**Salida esperada:**
```
hola mundo
```
.\tarea1 1_print.bytecode

### 2_aritmetrica.bytecode
def prueba2():
    print(7 + 5)    # 12
    print(10 - 4)   # 6
    print(6 * 7)    # 42
    print(7 // 2)   # 3 (división entera)
    print(7 % 2)    # 1
**Salida esperada (5 líneas):**
```
12
6
42
3
1
```
.\tarea1 2_aritmetrica.bytecode

### 3_logica.bytecode
def prueba3():
    print(True and False)  # False
    print(True or False)   # True
**Salida esperada:**
```
False
True
```
.\tarea1.exe 3_logica.bytecode

### 4_comparar_y_saltar.bytecode
def prueba4():
    x = 0
    while x < 5:
        print(x)
        x = x + 1
**Salida esperada (5 líneas):**
```
0
1
2
3
4
```
.\tarea1 4_comparar_y_saltar.bytecode

### 5_listas.bytecode
def prueba5():
    lista = [10, 20, 30]
    print(lista[1])   # 20
    lista[1] = 99
    print(lista[1])   # 99
**Salida esperada:**
```
20
99
```
.\tarea1 5_listas.bytecode 

### 6_pares.bytecode
def prueba6():
    x = 0
    lista = [0,1,2,3,4,5,6,7,8,9]
    while x < 10:
        if x % 2 == 0:
            print(lista[x])
        x = x + 1
**Salida esperada (5 líneas):**
```
0
2
4
6
8
```
.\tarea1 6_pares.bytecode 

---

    Casos de error

### 7_error_div.bytecode
def prueba7():
    # genera ZeroDivisionError
    print(7 // 0)
**Resultado esperado:** Terminación con mensaje de error: `división por cero`.

.\tarea1 7_error_div.bytecode