# Arriendos, pagos y comprobantes

El módulo conserva por separado la obligación esperada (`charges`), el dinero informado (`payments`) y su aplicación (`payment_allocations`). Esto permite completar un mes con uno, dos o múltiples abonos sin perder fechas, respaldos ni observaciones.

## Flujo mensual

1. Configurar una sola vez el arrendatario, la vigencia del contrato y la renta base.
   El arrendatario queda fijo durante esa vigencia; para cambiarlo se cierra el contrato con fecha y motivo y se crea uno nuevo.
2. Registrar por separado la obligación contractual: renta completa y día límite indicado en el contrato.
3. Solo cuando exista un acuerdo adicional, definir entre 2 y 12 cuotas como `special_agreement`. La suma debe coincidir con la renta, pero no reemplaza la obligación contractual original.
4. Al consultar un período comprendido por el contrato, crear automáticamente el cobro mensual y sus alertas de pago.
5. Agregar, cuando corresponda, luz, agua, basura o gastos comunes.
6. Registrar cada abono reutilizando el arrendatario del contrato y adjuntando fecha, monto, medio, referencia, observaciones y comprobante privado.
7. Revisar la notificación interna.
8. Confirmar manualmente el abono. El sistema nunca concilia automáticamente.
9. Cuando el saldo del arriendo llega a cero se emite un PDF inmutable con todas las cuotas confirmadas, firma electrónica simple y QR.
10. El QR abre una página pública con datos mínimos y el sello “Verificado por Control de Propiedades”.
11. La propiedad muestra un historial de vales con período, folio, descarga del PDF y verificación QR.

## Trazabilidad y análisis

- Los pagos confirmados no se editan; futuras reversiones deberán usar movimientos compensatorios.
- Las observaciones distinguen pagos, reportes del arrendatario, propiedad, servicios y notas generales.
- Los atrasos se clasifican comparando la fecha recibida con el vencimiento del cobro.
- Cada abono conserva el número de cuota y la fecha planificada que le correspondían.
- Los comprobantes bancarios se almacenan con confidencialidad restringida; los respaldos de servicios pueden clasificarse como confidenciales y verificarse con proveedor, fecha, período y monto.
- Los antecedentes de una corredora conservan por separado la garantía, la liquidación bancaria neta y la factura de comisión o administración. La vista de la propiedad reconstruye por período la renta bruta documentada (`depósito neto + descuento`) sin registrar esos descuentos como depósitos directos ni duplicar pagos ya conciliados.
- Los PDF e imágenes privados pueden visualizarse dentro de su ficha mediante una respuesta autenticada `inline`; la descarga original continúa disponible y ningún enlace vuelve público el archivo.
- El contrato conserva el aviso de renovación, la anticipación en meses y días para solicitar la restitución y el próximo reajuste por IPC. El reajuste siempre requiere revisión humana.
- Las condiciones de un contrato vigente se corrigen mediante una enmienda con fecha efectiva, motivo, estado anterior, estado nuevo y aprobación humana. La operación no permite cambiar al arrendatario ni reescribir períodos con pagos registrados.
- El calendario distingue la preparación anticipada de la renovación, la fecha máxima de aviso, la revisión IPC, su posible fecha de aplicación y la inspección de salida.
- Las mantenciones periódicas conservan categoría, frecuencia, próxima fecha, anticipación y última ejecución; al completarlas se calcula la siguiente fecha.
- El calendario trimestral parte en el mes seleccionado, prepara los cobros y alertas de los tres meses siguientes y combina alertas reales con sugerencias de revisión financiera, contractual, documental y de mantenciones.
- Las notificaciones usan claves de deduplicación para evitar alertas repetidas.
- Cada confirmación y emisión registra un evento de auditoría.
- Agentes futuros podrán leer y proponer análisis, pero no confirmar pagos.

La firma incluida es electrónica simple y trazable dentro de la plataforma. No se presenta como firma electrónica avanzada.
