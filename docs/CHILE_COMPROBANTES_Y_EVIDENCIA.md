# Criterios para comprobantes y evidencia de arriendo en Chile

Fecha de revisión: 28 de julio de 2026.

Este documento es una pauta técnica y documental. No reemplaza la revisión de un abogado para presentar una demanda, contestar una excepción, terminar el contrato o convenir una rebaja de renta.

## Base normativa revisada

- La [Ley 18.101](https://www.bcn.cl/leychile/navegar?idNorma=29526), artículo 23, reconoce expresamente el recibo de la renta; el artículo 21 regula reajustes en caso de mora. Sus artículos 18-A y siguientes exigen, para el procedimiento monitorio, identificar rentas y cuentas adeudadas, explicar con precisión su origen, forma, fecha y lugar, y acompañar todos los antecedentes de fundamento.
- La [Ley 19.799](https://www.bcn.cl/leychile/Navegar?idNorma=196640&r=1), artículos 3 y 5, reconoce los documentos y firmas electrónicas y permite presentar documentos electrónicos en juicio. Un instrumento privado con firma electrónica simple se aprecia conforme a las reglas generales; no equivale automáticamente a un instrumento público ni a una firma electrónica avanzada.
- El [artículo 348 bis del Código de Procedimiento Civil, incorporado por la Ley 20.217](https://www.bcn.cl/leychile/Navegar?idNorma=266348), contempla la audiencia de percepción documental y permite prueba complementaria de autenticidad si el documento electrónico es objetado.
- La [Ley 19.628 vigente hasta el 30 de noviembre de 2026](https://www.bcn.cl/leychile/Navegar?dt=open&idLey=19628) exige que el tratamiento de datos personales se ajuste a la ley y a finalidades permitidas. La reforma de la [Ley 21.719](https://www.bcn.cl/leychile/navegar?idNorma=141599&idParte=8642686&idVersion=2026-12-01) entra en vigor el 1 de diciembre de 2026 y refuerza las obligaciones, por lo que el sistema aplica desde ahora minimización, acceso restringido y trazabilidad.
- La necesidad de documento tributario depende de la calidad del arrendador y de las características del inmueble. El [Oficio SII 1962 de 2022](https://www4.sii.cl/gabineteAdmInternet/descargaArchivo?acc=download&extension=pdf&id=2525ff44-a1c2-478b-a562-4b961c9616f4&mediaType=application%2Fpdf&nombreDocumento=1962-23%2F06%2F2022.pdf) distingue, entre otros casos, al propietario persona natural de un inmueble no amoblado. Por eso el comprobante de la plataforma se rotula como constancia privada y no como boleta o factura.

## Reglas aplicadas al producto

1. Un comprobante final sólo se emite cuando el saldo de la renta del período llega a cero mediante pagos conciliados.
2. El PDF identifica folio, emisor, arrendador, arrendatario, propiedad sin dirección completa en la verificación pública, período, renta, total recibido, saldo, estado y detalle de abonos.
3. Un pago menor se registra como **abono parcial**. No se describe como renta pagada, descuento, compensación, condonación o modificación contractual sin un acuerdo separado y comprobable.
4. El QR utiliza una URL pública de producción. La verificación muestra estado, código, versión de plantilla y SHA-256 del PDF, sin revelar RUT ni datos bancarios.
5. La “firma electrónica simple” identifica quién emitió la constancia y cuándo. No se afirma que sea firma electrónica avanzada, certificación notarial ni instrumento público.
6. El comprobante advierte que es una constancia privada, no un documento tributario y que no modifica por sí solo el contrato.
7. Los comprobantes bancarios y conversaciones se guardan como originales, con nombre original, tamaño, tipo MIME, SHA-256, procedencia y evento de auditoría. No se reescriben ni se incrustan anotaciones en el original.
8. Las capturas de conversación se describen como capturas aportadas por el propietario. Para una controversia se recomienda preservar también el teléfono, realizar una exportación nativa completa del chat y conservar mensajes anteriores y posteriores, ya que una captura aislada puede ser objetada por autenticidad o falta de contexto.
9. El portal del arrendatario sólo expone contenido marcado expresamente como visible. Pagos, saldos, RUT, cuentas, archivos probatorios y notas internas quedan fuera.

## Campos mínimos del expediente de pago

- obligación y período;
- monto contractual, monto recibido y saldo;
- fecha efectiva y referencia bancaria;
- comprobante original con SHA-256;
- estado de conciliación y usuario que confirmó;
- observación sobre diferencias, sin inferir acuerdos;
- historial de auditoría no destructivo;
- comprobante final verificable cuando el saldo llega a cero.

## Antes de acciones legales

Solicitar revisión profesional del contrato, anexos, cronología completa, cálculo de reajustes/intereses, comunicaciones sobre reparaciones o habitabilidad, acreditación del dominio o representación, comprobantes de pago y saldo. No publicar la deuda ni los datos bancarios en el enlace compartido.
