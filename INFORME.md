
# 1. Protocolo de Comunicación

Para el intercambio de datos entre clientes (agencias) y el servidor se diseñó un **protocolo binario con codificación TLV (Type-Length-Value)** para los campos de longitud variable (nombres y apellidos). Los campos de tamaño fijo —tales como documento, fecha de nacimiento y número de apuesta— cuentan con una longitud de bytes predefinida.

> **Importante:** Todos los campos numéricos enteros son codificados en orden **Big-Endian** previo a su transmisión por la red para evitar inconsistencias de arquitectura (*endianness*) entre el cliente y el servidor.

## 1.1. Primera Implementación: Envío Individual de Apuestas

En la primera versión del sistema, la comunicación se realizaba mediante el envío de apuestas individuales.

### Tipos de mensajes

```text
[Apuesta]            --->  (Cliente envía al Servidor una apuesta individual)
[Fin de apuestas]    --->  (Cliente notifica al Servidor el fin de transmisión)
[Lista de Ganadores] --->  (Servidor responde al Cliente con los ganadores del sorteo)
```

### A. Envio de Apuesta (`SendBet` ---> `receive_bet`)

El cliente serializa los campos numéricos en formato binario e introduce un byte indicador de longitud previo a cada cadena variable.

| Campo | Tamaño (Bytes) | Formato / Codificación | Descripción |
| :--- | :--- | :--- | :--- |
| `agency_id` | 2 bytes | UInt16 (Big-Endian) | Identificador numérico de la agencia. |
| `name_length` | 1 byte | UInt8 | Longitud ($N$) del nombre del apostador. |
| `first_name` | $N$ bytes | UTF-8 String | Cadena de caracteres con el nombre. |
| `last_name_length` | 1 byte | UInt8 | Longitud ($M$) del apellido del apostador. |
| `last_name` | $M$ bytes | UTF-8 String | Cadena de caracteres con el apellido. |
| `document` | 4 bytes | UInt32 (Big-Endian) | Número de documento (DNI). |
| `birthdate` | 10 bytes | ASCII String (`YYYY-MM-DD`) | Fecha de nacimiento fija en texto de 10 bytes. |
| `number` | 4 bytes | UInt32 (Big-Endian) | Número apostado al sorteo. |

#### Reglas del Protocolo:

1. Las cadenas de first_name y last_name no deben superar los 255 bytes ($N, M \le 255$) para no desbordar el límite de un entero de 1 byte (UInt8).
2. Los valores numéricos document y number están delimitados por el límite máximo de un entero sin signo de 32 bits ($2^{32}-1$).
3. En esta primera versión, un nombre con longitud 0 era interpretado por el servidor como señal centinela de finalización.

### B. Fin de Apuestas (`SendEnd` ---> `receive_bet`)

El cliente notifica al servidor que la agencia ha finalizado la transferencia de todas sus apuestas para dar paso a la ejecución del sorteo.

| Campo | Tamaño (Bytes) | Valor / Contenido | Descripción |
| :--- | :--- | :--- | :--- |
| `agency_id` | 2 bytes | UInt16 (Big-Endian) | Identificador de la agencia que finaliza. |
| `name_length` | 1 byte | `0x00` | Byte nulo. Actúa como señal especial. |

Para señalar que una agencia había finalizado la transferencia de todas sus apuestas, se enviaba un mensaje especial donde el campo name_length tenía el valor `0x00`.

### C. Lista de Ganadores (`send_winners` ---> `ReceiveWinners`)

El servidor envía un mensaje con la totalidad de los ganadores del sorteo a las agencias.

**Encabezado / Conteo de Ganadores**

| Campo | Tamaño (Bytes) | Formato / Codificación | Descripción |
| :--- | :--- | :--- | :--- |
| `winners_count` | 4 bytes | UInt32 (Big-Endian) | Cantidad de ganadores ($K$) que se enviarán a continuación. |

---

**Registro de Ganador (se repite $K$ veces)**

Estructura idéntica a la [Tabla de Envío de Apuesta Individual](#a-envio-de-apuesta-`sendbet`-----`receive_bet`), omitiendo el campo inicial `agency_id`.


## 1.2. Segunda Implementación (Versión Final): Procesamiento por Lotes (Batching)

Con el fin de responder a los requerimientos de la prueba N°6 —donde los clientes deben empaquetar múltiples apuestas en un mismo mensaje de tamaño variable (`BATCH_SIZE`) y recibir confirmación del servidor solo tras la correcta recepción del lote completo— se rediseñó el protocolo de red.

Se añadió un encabezado global al paquete compuesto por el identificador de la agencia y la cantidad de registros contenidos en el lote.

### Tipos de mensajes

```text
[Lote de Apuestas]   --->  (Cliente envía lote de múltiples apuestas al Servidor)
[ACK de Lote]        <---  (Servidor confirma recepción exitosa del lote al Cliente)
[Fin de Apuestas]    --->  (Cliente envía señal de fin de transmisión al Servidor)
[Lista de Ganadores] <---  (Servidor responde al Cliente con los ganadores)
```

### A. Envío de Lote de Apuestas (`SendBatch` ---> `receive_batch`)

El cliente envía un encabezado fijo indicando la cantidad de apuestas que componen el paquete:

| Campo | Tamaño (Bytes) | Formato / Codificación | Descripción |
| :--- | :--- | :--- | :--- |
| `agency_id` | 2 bytes | UInt16 (Big-Endian) | Identificador numérico de la agencia. |
| `batch_SIZE` | 2 bytes | UInt16 (Big-Endian) | Cantidad de apuestas ($B$) adjuntas en el lote. |

---

A continuación del encabezado, se adjuntan en serie las $B$ apuestas con la estructura descrita en [Tabla de Envío de Apuesta Individual](#a-envio-de-apuesta-`sendbet`-----`receive_bet`) (sin repetir `agency_id`). 


### B. Confirmación de Recepción de Lote (ACK: dentro de `receive_batch` ---> llega a `SendBatch`)

Tras recibir y procesar correctamente el lote, el servidor responde con un mensaje de acuse de recibo:

- Payload: `b"\x00"` (1 byte con valor 0x00), confirmando que el batch ha sido aceptado con éxito.

### C. Fin de Apuestas (`SendEnd` ---> `receive_batch`)

Se simplificó la señalización de cierre de transmisión:

1. El cliente envía un paquete de encabezado con `batch_size = 0` y su respectivo `agency_id`.
2. Al recibir un paquete de tamaño cero, el servidor interpreta el término de transmisión de la agencia, devuelve un ACK final de confirmación y procede al cierre ordenado de la conexión.


### D. Lista de Ganadores

Mantiene el mismo formato que en la versión inicial (encabezado de conteo $K$ seguido por $K$ registros equivalentes a la [Tabla de Envío de Apuesta Individual](#a-envio-de-apuesta-`sendbet`-----`receive_bet`) sin `agency_id`).

## 1.3. Optimizaciones de Memoria

Para garantizar que la aplicación pudiese superar las pruebas de volumen masivo de datos sin incurrir en fallos por saturación de recursos (out-of-memory o presión excesiva sobre el Garbage Collector):

- Preasignación de Buffers (`SendBatch`): Se implementó un algoritmo de dos pasadas sobre la lista de apuestas a enviar.

    1. Primera pasada: Calcula la suma exacta en bytes que requerirá el lote serializado (`totalSize`).

    2. Reserva única: Se aloca el slice de memoria de manera contigua una sola vez utilizando `make([]byte, headerSize, totalSize)`.

Con esto se eliminaron las realocaciones dinámicas repetitivas por cada apuesta individual, logrando superar con éxito los tests de volumen.

# 2. Mecanismos para Sincronizar la Ejecución Concurrente

Para desacoplar la aceptación de conexiones de la lógica de negocio, el servidor implementa un modelo Thread-per-Client. Cada vez que una agencia establece una conexión de red, el hilo principal instancia y lanza un objeto `ClientHandle` (el cual hereda de `threading.Thread`). De esta forma, cada agencia es atendida por un hilo dedicado que procesa su flujo de mensajes de manera independiente y paralela.

Dado que estos hilos comparten recursos globales y deben coordinar sus etapas de ejecución, se diseñaron y evolucionaron los mecanismos de sincronización para abordar adecuadamente la concurrencia.

## 2.1 Evolución de los Mecanismos de Sincronización

### 2.1.1 Versión Inicial: Mutex Único (`threading.Lock`) y Barrera (`threading.Barrier`)

En la primera versión del diseño, la sincronización se apoyaba en dos primitivas estándar de Python:
- **Exclusión Mutua (`threading.Lock`):** Se utilizaba un único Lock exclusivo tanto para escribir lotes de apuestas (`store_bets`) como para leer el archivo completo durante la determinación de ganadores (`load_bets`).
- **Barrera de Sincronización (`threading.Barrier`):** Se configuraba con el quórum mínimo de agencias (`AGENCY_QUORUM_MIN`). Cada hilo ejecutaba `barrier.wait()` al finalizar la transmisión para aguardar a que el quórum se alcanzara antes de iniciar el sorteo.

#### Limitaciones identificadas en la Versión Inicial:
1. **Comportamiento Cíclico de la Barrera:** La primitiva `threading.Barrier` se restablece automáticamente tras ser superada por N hilos. Si el quórum mínimo era N=5 y se conectaban N=6 agencias, la 6ta agencia alcanzaba la barrera cuando esta ya se había reiniciado, quedando atrapada en espera indefinida (deadlock) aguardando 4 agencias adicionales que nunca llegarían.
2. **Serialización Innecesaria de Lecturas:** El `Lock` exclusivo serializaba las operaciones de `load_bets()`. Una vez finalizado el registro de apuestas, múltiples hilos intentando consultar ganadores de forma simultánea debían esperar secuencialmente su turno para leer el archivo en disco, desperdiciando la capacidad de paralelizarlas.

---

### 2.1.2 Versión Final: `threading.Event` + Contador Atómico y `ReadWriteLock` Personalizado

Para subsanar las limitaciones anteriores y maximizar el paralelismo en la fase de consulta, se rediseñó el esquema utilizando dos mecanismos avanzados:

1. **Quórum de Apertura Única (`threading.Event` + Contador Atómico):**
   - Se reemplazó la barrera por un contador compartido protegido por un `threading.Lock` (`agency_counter`) y un evento `threading.Event` (`lottery_ready_event`).
   - Cada hilo incrementa el contador al terminar de recibir apuestas. El hilo que alcanza o supera el quórum mínimo activa el evento mediante `lottery_ready_event.set()`.
   - **Ventaja:** Al activarse, el `Event` permanece abierto en estado señalizado permanentemente. Si una agencia posterior termina sus envíos tras haber superado el quórum, al invocar `lottery_ready_event.wait()` pasa inmediatamente sin bloquearse.

2. **Cerrojo de Lectores/Escritor (`ReadWriteLock`):**
   - Se implementó un cerrojo a medida (ubicado en `rw_lock.py`) basado en `threading.Condition`.
   - **Escrituras Exclusivas (`acquire_write` / `release_write`):** Durante la recepción e inserción de apuestas (`store_bets`), el hilo adquiere el cerrojo de forma exclusiva, bloqueando tanto a otros escritores como a lectores.
   - **Lecturas Concurrentes (`acquire_read` / `release_read`):** Durante la consulta de ganadores (`load_bets`), múltiples hilos adquieren el cerrojo en modo lectura en simultáneo. El lock interno de la condición sólo se retiene durante una fracción de milisegundo para incrementar/decrementar el contador de lectores activos (`_readers`), permitiendo que el I/O de lectura sobre el archivo se ejecute de forma paralela entre todas las agencias.

---

## 2.2 Flujo de Ejecución del Hilo `ClientHandle`

El ciclo de vida de cada hilo de atención se divide en cuatro fases secuenciales:

1. **Fase de Recepción e Inserción Concurrente:**
   - El hilo lee lotes de apuestas desde el socket de la agencia en un bucle.
   - Para cada lote válido, adquiere el cerrojo en modo escritura exclusiva (`rw_lock.acquire_write()`), persiste las apuestas en disco (`store_bets`) y libera el cerrojo inmediatamente (`rw_lock.release_write()`).

2. **Fase de Sincronización por Quórum:**
   - Al finalizar la transmisión de la agencia, el hilo incrementa de forma atómica el contador global de agencias finalizadas (`agency_counter`).
   - Si el contador alcanza o supera `AGENCY_QUORUM_MIN`, se gatilla `lottery_ready_event.set()`.

3. **Fase de Espera de Sorteo:**
   - El hilo invoca `lottery_ready_event.wait()`, suspendiéndose únicamente si el quórum aún no ha sido alcanzado. Si el quórum ya fue alcanzado previa o simultáneamente, transiciona sin demora.

4. **Fase de Consulta Concurrente y Notificación de Ganadores:**
   - Adquiere el cerrojo en modo lectura concurrente (`rw_lock.acquire_read()`) para consultar las apuestas consolidadas (`load_bets()`).
   - Múltiples hilos leen en disco simultáneamente. Cada hilo filtra los ganadores de su propia agencia (`agency_id`), envía los resultados por el socket (`send_winners`) y libera el acceso de lectura (`rw_lock.release_read()`) antes de cerrar la conexión de forma limpia.


# 3. Librerías Incorporadas en la Solución Final

## 3.1. Nuevas Librerías en Python (Servidor)
- `threading`: Incorporado para posibilitar el esquema de concurrencia multihilo y proveer los mecanismos de sincronización (`Lock`, `Barrier` / `Condition`). (Valida para ultilizar, confirmada en el foro)

- `signal`: Utilizado para la captura e interrupción limpia del servidor (graceful shutdown) ante señales del sistema operativo (`SIGTERM`). (Valida para ultilizar, confirmada en el foro)

- `.clientHandle` / `.rw_lock` / `ServerProtocol`: Módulos propios introducidos para encapsular la lógica de interacción individual por socket, proveer el cerrojo de lectura/escritura (Readers-Writer Lock) y abstraer el parseo de mensajes del protocolo TLV por lotes.

## 3.2. Nuevas Librerías en Go (Cliente)

- `bytes`: Fundamental para la manipulación eficiente de buffers contiguos de bytes (`bytes.Buffer`) y para la pre-asignación de memoria de lotes sin generar reasignaciones en el Heap.

- `encoding/binary`: Utilizado para la serialización de enteros (`binary.BigEndian.PutUint16, binary.BigEndian.PutUint32`) ajustados al protocolo binario en red. (Valida para ultilizar para BigEndian y LittleEndian, confirmada en el foro)

- `bufio`: Incorporado para la lectura e inspección de streams de datos con buffering, optimizando la lectura sobre archivos. (Valida para ultilizar, confirmada en el foro)

- `strconv`: Utilizado en el parseo y conversión de tipos numéricos a representaciones en cadena durante el armado de registros y lectura de configuraciones.

- `syscall` / `os/signal`: Implementados para el manejo y canalización de interrupciones del sistema operativo a nivel de proceso en el cliente. (Validas para ultilizar, confirmada en el foro)