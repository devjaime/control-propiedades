# Prompt funcional — Control Propiedades

## Instrucción para Codex

Actúa como **product engineer senior, analista funcional y diseñador de dominio**. Debes ayudarme a construir de forma incremental una aplicación web denominada provisionalmente **Control Propiedades**, destinada inicialmente a administrar mis propias propiedades y preparada desde su diseño para evolucionar en una segunda versión hacia un SaaS de administración inmobiliaria con un **corredor virtual asistido por agentes de IA**.

No implementes todo de una sola vez. Trabaja por iteraciones pequeñas, verificables y desplegables. Antes de modificar código:

1. Inspecciona el repositorio completo.
2. Resume la arquitectura actual, dependencias, convenciones y deuda técnica.
3. Identifica qué parte de esta especificación ya está implementada.
4. Propón un plan de trabajo de máximo 5 tareas.
5. Ejecuta solo la primera tarea o el alcance que yo indique.
6. Incluye pruebas, migraciones, documentación y criterios de aceptación.
7. No elimines código ni datos existentes sin explicar el impacto.
8. No agregues microservicios salvo que exista una necesidad objetiva y documentada.

---

# 1. Visión del producto

Construir un sistema web que centralice la administración operativa, financiera, documental y de mantenimiento de una o más propiedades.

La primera propiedad administrada será una casa arrendada en Coquimbo. También debe permitir administrar mi vivienda principal y futuras propiedades.

El sistema debe poder responder en cualquier momento:

- Qué propiedades existen.
- Quiénes son sus propietarios.
- Cuál es su uso actual.
- Qué contratos están vigentes.
- Qué documentos existen y cuáles faltan.
- Qué cobros debían realizarse.
- Qué pagos fueron recibidos.
- Qué comprobantes fueron emitidos.
- Qué obligaciones están pendientes.
- Qué dividendos y contribuciones fueron pagados.
- Qué reparaciones, mantenciones y mejoras se ejecutaron.
- Cuánto cuesta, produce o debe cada propiedad.
- Qué acciones requieren revisión humana.
- Qué acciones fueron ejecutadas por usuarios o agentes.

---

# 2. Principios funcionales

## 2.1 Fuente oficial de información

La base de datos es la fuente oficial del estado del sistema.

Los documentos son evidencia asociada a entidades estructuradas, pero subir un documento no debe cambiar automáticamente un estado financiero o contractual sin validación.

Ejemplo:

- Un comprobante bancario subido no debe marcar un arriendo como pagado.
- Primero debe relacionarse con una propiedad, contrato, período, cobro esperado, monto y fecha.
- Después debe pasar por conciliación.
- Finalmente un usuario autorizado confirma el pago y permite emitir el comprobante.

## 2.2 Trazabilidad

Toda acción relevante debe registrar:

- Usuario o agente que la ejecutó.
- Fecha y hora.
- Entidad afectada.
- Estado anterior.
- Estado nuevo.
- Motivo o comentario.
- Dirección IP o contexto técnico cuando corresponda.
- Identificador de ejecución del agente cuando corresponda.

## 2.3 Aprobación humana

Las siguientes acciones deben requerir aprobación humana:

- Confirmar un pago.
- Anular o reemplazar un comprobante.
- Modificar un contrato vigente.
- Cambiar montos ya conciliados.
- Eliminar documentos.
- Aprobar gastos o trabajos.
- Enviar comunicaciones legales.
- Ejecutar acciones financieras.
- Cambiar permisos de usuarios.
- Autorizar acciones propuestas por agentes.

## 2.4 SaaS-ready desde la primera versión

Aunque la primera versión será para uso personal, el modelo debe incorporar desde el comienzo:

- Organización o cuenta.
- Usuarios.
- Roles.
- Propiedades pertenecientes a una organización.
- Separación lógica de datos por organización.
- Auditoría.
- Planes o límites preparados, aunque no se cobre aún.
- Configuración por organización.
- Posibilidad futura de dominios personalizados.
- Posibilidad futura de múltiples administradores y corredores.

---

# 3. Perfiles de usuario

## 3.1 Superadministrador de plataforma

Reservado para una futura versión SaaS.

Puede:

- Administrar organizaciones.
- Revisar métricas globales.
- Suspender cuentas.
- Gestionar planes y límites.
- Revisar eventos de seguridad.
- Acceder solo mediante mecanismos auditados de soporte.

## 3.2 Administrador de organización

Puede:

- Administrar usuarios y roles.
- Crear y modificar propiedades.
- Registrar contratos.
- Confirmar pagos.
- Emitir comprobantes.
- Administrar obligaciones.
- Aprobar trabajos.
- Configurar agentes e integraciones.
- Exportar información.

## 3.3 Gestor de propiedad

Puede:

- Ver propiedades asignadas.
- Registrar documentos.
- Registrar pagos pendientes de aprobación.
- Gestionar mantenciones.
- Comunicarse con arrendatarios.
- No puede modificar configuración crítica ni usuarios.

## 3.4 Propietario o copropietario

Puede:

- Consultar información financiera y documental.
- Ver reportes.
- Aprobar determinadas acciones.
- Tener acceso limitado a propiedades específicas.

## 3.5 Arrendatario

Disponible en una versión posterior.

Puede:

- Consultar sus contratos.
- Ver cobros.
- Subir comprobantes.
- Descargar comprobantes emitidos.
- Reportar problemas.
- Consultar estado de solicitudes.
- No puede acceder a información financiera interna del propietario.

## 3.6 Agente de IA

Es una identidad técnica con permisos explícitos.

Puede:

- Leer información autorizada.
- Proponer clasificaciones.
- Proponer conciliaciones.
- Crear borradores.
- Detectar vencimientos.
- Generar resúmenes.
- Crear solicitudes de aprobación.

No puede:

- Confirmar pagos.
- Eliminar documentos.
- Modificar contratos vigentes.
- Enviar comunicaciones legales.
- Ejecutar pagos.
- Cambiar permisos.
- Acceder a secretos o credenciales humanas.

---

# 4. Entidades funcionales

## 4.1 Organización

Representa al dueño lógico de los datos.

Campos principales:

- Nombre.
- Identificador.
- Estado.
- Zona horaria.
- Moneda predeterminada.
- País.
- Configuración tributaria.
- Configuración de notificaciones.
- Límites del plan.
- Fecha de creación.

## 4.2 Usuario

Campos principales:

- Nombre.
- Correo.
- Estado.
- Rol.
- Organización.
- Último acceso.
- Autenticación multifactor activada o no.
- Propiedades asignadas.
- Preferencias de notificación.

## 4.3 Propiedad

Campos principales:

- Organización.
- Nombre interno.
- Tipo.
- Uso.
- Estado.
- Dirección.
- Comuna.
- Región.
- País.
- Rol de avalúo.
- Fecha de compra.
- Valor de compra.
- Moneda o UF.
- Tasación inicial.
- Tasaciones posteriores.
- Superficie construida.
- Superficie de terreno.
- DFL2.
- Amoblada o no.
- Porcentaje de propiedad.
- Estado hipotecario.
- Cuenta bancaria receptora asociada mediante referencia segura.
- Fotografías.
- Notas.

Usos soportados:

- Vivienda principal.
- Propiedad arrendada.
- Segunda vivienda.
- Vacante.
- En remodelación.
- En venta.
- En adquisición.

## 4.4 Propietario y copropiedad

Permitir:

- Uno o más propietarios.
- Porcentaje de participación.
- Fecha de inicio.
- Fecha de término.
- Tipo de titularidad.
- Documentos de respaldo.

## 4.5 Arrendatario

Campos principales:

- Nombre.
- RUT o identificador.
- Correo.
- Teléfono.
- Dirección de contacto.
- Contacto de emergencia.
- Estado.
- Documentos relacionados.
- Consentimientos y registros de comunicaciones.

Los datos personales deben protegerse y minimizarse.

## 4.6 Contrato de arriendo

Campos principales:

- Propiedad.
- Arrendatarios.
- Fecha de inicio.
- Fecha de término.
- Estado.
- Renta base.
- Moneda o UF.
- Día de pago.
- Mecanismo de reajuste.
- Periodicidad del reajuste.
- Garantía.
- Responsable de servicios.
- Inventario de entrega.
- Condiciones especiales.
- Documento firmado.
- Anexos.
- Renovaciones.
- Historial de cambios.

Estados sugeridos:

- Borrador.
- Pendiente de firma.
- Vigente.
- Próximo a vencer.
- Renovado.
- Terminado.
- Rescindido.
- Archivado.

## 4.7 Cobro

Representa una obligación de pago esperada.

Tipos:

- Arriendo.
- Reajuste.
- Servicio reembolsable.
- Gasto común.
- Multa.
- Interés.
- Reparación imputable.
- Saldo anterior.
- Otro.

Campos:

- Contrato.
- Propiedad.
- Arrendatario.
- Período.
- Fecha de vencimiento.
- Monto.
- Moneda.
- Estado.
- Saldo pendiente.
- Concepto.
- Origen automático o manual.

Estados:

- Borrador.
- Pendiente.
- Parcialmente pagado.
- Pagado.
- Vencido.
- Anulado.
- En disputa.

## 4.8 Pago

Representa dinero efectivamente recibido.

Campos:

- Propiedad.
- Contrato.
- Pagador.
- Fecha.
- Monto.
- Moneda.
- Medio de pago.
- Referencia bancaria.
- Cuenta receptora.
- Documento de respaldo.
- Estado de conciliación.
- Observaciones.
- Usuario o agente que lo registró.
- Usuario que lo confirmó.

Estados:

- Recibido.
- Pendiente de conciliación.
- Conciliado.
- Rechazado.
- Revertido.
- Duplicado.

## 4.9 Aplicación de pago

Relaciona un pago con uno o más cobros.

Debe soportar:

- Pagos parciales.
- Un pago aplicado a varios cobros.
- Varios pagos aplicados a un cobro.
- Saldos a favor.
- Reversiones.
- Ajustes auditados.

## 4.10 Comprobante de arriendo

Documento privado emitido después de confirmar un pago.

Debe incluir:

- Folio único.
- Organización emisora.
- Propiedad con información limitada.
- Arrendador.
- Arrendatario con datos parcialmente ocultos.
- Período.
- Conceptos pagados.
- Fecha de recepción.
- Monto total.
- Medio de pago.
- Estado.
- Código de verificación.
- QR.
- Hash del PDF.
- Fecha de emisión.

Estados:

- Vigente.
- Reemplazado.
- Anulado.

Reglas:

- No se edita un comprobante emitido.
- Para corregirlo se reemplaza.
- Debe conservarse el original.
- El QR apunta a una página pública de validación.
- La página pública no expone RUT completo, cuenta bancaria, contrato ni dirección detallada.
- El documento debe poder descargarse nuevamente.

## 4.11 Documento

Tipos iniciales:

- Escritura.
- Inscripción de dominio.
- Certificado DFL2.
- Tasación.
- Contrato.
- Anexo.
- Inventario.
- Comprobante bancario.
- Comprobante de arriendo.
- Dividendo.
- Contribución.
- Seguro.
- Gasto común.
- Servicio básico.
- Presupuesto.
- Factura.
- Boleta.
- Garantía.
- Fotografía.
- Informe de inspección.
- Comunicación.
- Certificado.
- Otro.

Metadatos mínimos:

- Organización.
- Propiedad.
- Tipo.
- Nombre original.
- Nombre normalizado.
- Fecha documental.
- Período.
- Emisor.
- Monto.
- Moneda.
- Tipo MIME.
- Tamaño.
- Hash SHA-256.
- Ubicación de almacenamiento.
- Estado.
- Etiquetas.
- Datos extraídos.
- Nivel de confidencialidad.
- Fecha de carga.
- Usuario o agente que lo cargó.
- Fecha de verificación.

Estados:

- Cargado.
- Procesando.
- Clasificado.
- Pendiente de revisión.
- Verificado.
- Rechazado.
- Duplicado.
- Archivado.

## 4.12 Obligación

Tipos:

- Contribuciones.
- Dividendo hipotecario.
- Seguro.
- Renovación de seguro.
- Servicio básico.
- Gasto común.
- Revisión contractual.
- Reajuste de renta.
- Inspección.
- Mantención preventiva.
- Garantía.
- Certificado.
- Declaración anual.
- Otro.

Debe permitir:

- Frecuencia.
- Fecha de vencimiento.
- Responsable.
- Recordatorios.
- Documento requerido.
- Pago asociado.
- Estado.
- Repetición.
- Historial.

## 4.13 Crédito hipotecario

Campos:

- Propiedad.
- Institución.
- Número de operación parcialmente oculto.
- Fecha de inicio.
- Plazo.
- Moneda o UF.
- Monto original.
- Tasa.
- Dividendo.
- Día de pago.
- Saldo informado.
- Documentos.
- Estado.

## 4.14 Gasto

Clasificaciones:

- Gasto operativo.
- Mantención preventiva.
- Mantención correctiva.
- Reparación de emergencia.
- Mejora de capital.
- Daño del arrendatario.
- Trabajo por garantía.
- Gasto administrativo.
- Otro.

## 4.15 Orden de trabajo

Campos:

- Propiedad.
- Problema.
- Prioridad.
- Categoría.
- Responsable.
- Proveedor.
- Presupuesto.
- Costo real.
- Fechas.
- Fotografías antes y después.
- Documentos.
- Garantía.
- Impacto esperado.
- Aprobaciones.
- Estado.

Estados:

- Reportado.
- En evaluación.
- Presupuestado.
- Pendiente de aprobación.
- Aprobado.
- Programado.
- En ejecución.
- Terminado.
- Verificado.
- Cancelado.

## 4.16 Proveedor

Campos:

- Nombre.
- Especialidad.
- Contactos.
- Cobertura geográfica.
- Documentos.
- Historial de trabajos.
- Evaluación.
- Estado.

## 4.17 Evento de auditoría

Debe registrar todos los cambios críticos.

## 4.18 Ejecución de agente

Debe registrar:

- Agente.
- Modelo.
- Versión de prompt.
- Herramienta.
- Fecha.
- Entrada.
- Salida.
- Tokens cuando estén disponibles.
- Costo cuando esté disponible.
- Acciones propuestas.
- Acciones aprobadas.
- Errores.
- Correlación con entidades del sistema.

---

# 5. Flujos principales

## 5.1 Registrar una propiedad

1. El administrador crea la propiedad.
2. Registra propietarios.
3. Registra crédito hipotecario si existe.
4. Carga documentos iniciales.
5. Configura obligaciones.
6. Define si es vivienda principal o arrendada.
7. El sistema muestra información faltante.

## 5.2 Cargar un documento

1. El usuario selecciona propiedad.
2. Sube imagen o PDF.
3. El sistema calcula hash.
4. Detecta duplicados.
5. Guarda el archivo.
6. Crea un registro documental.
7. Opcionalmente envía el documento a extracción con Gemini.
8. La IA propone tipo, fecha, monto, emisor y período.
9. El usuario revisa.
10. El documento queda verificado.
11. El documento puede vincularse a contratos, pagos, obligaciones o trabajos.

## 5.3 Registrar un contrato

1. Crear borrador.
2. Asociar propiedad y arrendatarios.
3. Definir condiciones.
4. Cargar contrato y anexos.
5. Validar campos obligatorios.
6. Activar contrato.
7. Generar calendario de cobros.
8. Crear obligaciones de reajuste y vencimiento.

## 5.4 Cobro mensual de arriendo

1. Un proceso programado genera el cobro del período.
2. El sistema calcula reajustes cuando corresponda.
3. El cobro queda pendiente.
4. Se puede enviar recordatorio.
5. Si vence, cambia a vencido.
6. El historial debe conservarse.

## 5.5 Recepción de pago

1. El arrendatario envía un comprobante.
2. El usuario lo sube o un agente lo ingresa.
3. Se crea el pago pendiente.
4. Se extraen datos.
5. Se buscan cobros compatibles.
6. Se propone conciliación.
7. El administrador revisa.
8. Confirma el pago.
9. Se actualizan saldos.
10. Se genera el comprobante.
11. Se registra auditoría.

## 5.6 Emisión de comprobante con QR

1. Solo se emite desde un pago confirmado.
2. Se genera folio.
3. Se genera código público aleatorio.
4. Se produce el PDF.
5. Se calcula hash.
6. Se guarda como documento.
7. Se crea la URL de validación.
8. Se envía o descarga.
9. La validación pública muestra información limitada.
10. Un comprobante reemplazado muestra su estado y referencia al nuevo, sin exponer datos privados.

## 5.7 Registrar contribuciones

1. Crear obligación.
2. Registrar vencimiento.
3. Adjuntar aviso o certificado.
4. Registrar pago.
5. Adjuntar comprobante.
6. Verificar.
7. Marcar período como pagado.
8. Mostrar siguiente vencimiento.

## 5.8 Registrar dividendos

1. Crear obligación mensual.
2. Registrar monto esperado.
3. Cargar comprobante.
4. Confirmar.
5. Actualizar historial.
6. Registrar saldo hipotecario cuando exista información.

## 5.9 Mantenimiento

1. Registrar incidente.
2. Adjuntar fotografías.
3. Clasificar urgencia.
4. Solicitar presupuestos.
5. Aprobar proveedor.
6. Registrar ejecución.
7. Cargar facturas.
8. Registrar costo final.
9. Adjuntar fotografías finales.
10. Registrar garantía.
11. Clasificar como gasto o mejora de capital.

## 5.10 Administración de usuarios

1. El administrador invita usuario.
2. Asigna rol.
3. Asigna propiedades.
4. Define permisos.
5. El usuario activa cuenta.
6. Se registra aceptación.
7. Cambios de permisos quedan auditados.
8. Debe existir desactivación y revocación de sesiones.

---

# 6. Paneles y vistas

## 6.1 Dashboard general

Mostrar:

- Número de propiedades.
- Cobros del mes.
- Pagos recibidos.
- Pagos pendientes.
- Cobros vencidos.
- Obligaciones próximas.
- Documentos pendientes de revisión.
- Trabajos abiertos.
- Alertas de contratos.
- Acciones propuestas por agentes.
- Flujo mensual agregado.

## 6.2 Vista de propiedad

Pestañas:

- Resumen.
- Datos generales.
- Propietarios.
- Documentos.
- Contratos.
- Cobros.
- Pagos.
- Comprobantes.
- Obligaciones.
- Hipoteca.
- Gastos.
- Mantenciones.
- Mejoras.
- Fotografías.
- Auditoría.
- Agentes.

## 6.3 Bandeja documental

Debe permitir:

- Arrastrar y soltar.
- Cámara desde teléfono.
- Carga múltiple.
- Vista previa.
- Filtrado.
- Etiquetas.
- Detección de duplicados.
- Revisión de datos extraídos.
- Vinculación a entidades.

## 6.4 Bandeja de aprobaciones

Mostrar:

- Pagos por confirmar.
- Documentos por verificar.
- Conciliaciones propuestas.
- Gastos por aprobar.
- Acciones de agentes.
- Cambios contractuales.
- Comprobantes por anular o reemplazar.

## 6.5 Vista móvil

Debe permitir como mínimo:

- Iniciar sesión.
- Ver alertas.
- Fotografiar y subir documentos.
- Fotografiar problemas.
- Consultar propiedad.
- Confirmar o rechazar una propuesta.
- Descargar comprobantes.
- Ver estado de pagos y obligaciones.

---

# 7. Métricas

## Propiedad arrendada

- Renta esperada.
- Renta recibida.
- Saldo pendiente.
- Días de atraso.
- Ocupación.
- Gastos operativos.
- Mantenciones.
- Mejoras de capital.
- Flujo antes de financiamiento.
- Dividendos.
- Flujo después de financiamiento.
- Rentabilidad sobre capital aportado.
- Próximos vencimientos.
- Documentos faltantes.

## Vivienda principal

- Costo mensual.
- Dividendos pagados.
- Saldo hipotecario registrado.
- Contribuciones.
- Seguros.
- Mantenciones.
- Mejoras.
- Evolución de tasaciones.
- Costo patrimonial anual.

---

# 8. Agentes e IA

## 8.1 Integraciones previstas

- OpenClaw local.
- Hermes Agents local.
- Modelos locales como MiniMax 2.7.
- Gemini para lectura de imágenes y PDF.
- Otros proveedores mediante adaptadores.

## 8.2 Agentes iniciales

### Agente documental

- Recibe un archivo.
- Identifica tipo.
- Propone propiedad.
- Extrae fecha, monto, emisor y período.
- Detecta posibles duplicados.
- Crea una propuesta para revisión.

### Agente de conciliación

- Compara pagos con cobros.
- Propone asignaciones.
- Detecta diferencias.
- No confirma pagos.

### Agente de obligaciones

- Revisa vencimientos.
- Detecta documentos faltantes.
- Propone recordatorios.
- No inicia sesión en sitios externos con credenciales humanas.

### Agente de mantenimiento

- Clasifica incidentes.
- Propone prioridad.
- Organiza documentos.
- Resume presupuestos.
- No adjudica trabajos.

### Agente de portafolio

- Resume flujo.
- Detecta anomalías.
- Compara propiedades.
- Prepara reportes.

## 8.3 Corredor virtual

En una versión posterior debe poder:

- Recordar pagos.
- Recibir comprobantes.
- Entregar comprobantes emitidos.
- Gestionar solicitudes.
- Coordinar inspecciones.
- Preparar renovaciones.
- Mantener bitácora de comunicaciones.
- Escalar a humano cuando exista riesgo contractual, legal o financiero.

---

# 9. Reglas de privacidad y seguridad

- Los archivos son privados por defecto.
- El QR expone información mínima.
- El RUT debe mostrarse parcialmente oculto.
- No guardar claves bancarias, del SII o TGR.
- No registrar secretos en logs.
- Las descargas privadas deben usar autorización o URLs temporales.
- Todos los cambios críticos deben auditarse.
- Debe existir respaldo y restauración.
- El usuario debe poder exportar sus datos.
- Preparar políticas de conservación y eliminación.
- Separar datos por organización.
- Aplicar principio de mínimo privilegio.

---

# 10. Alcance del MVP

El MVP queda completo cuando se puede:

1. Crear una organización.
2. Crear usuarios y roles.
3. Registrar una propiedad.
4. Registrar propietarios.
5. Cargar y clasificar documentos.
6. Crear arrendatario.
7. Crear contrato.
8. Generar cobro mensual.
9. Subir comprobante bancario.
10. Registrar pago.
11. Conciliarlo manualmente.
12. Confirmar pago.
13. Emitir comprobante PDF.
14. Validarlo mediante QR.
15. Registrar contribuciones.
16. Registrar dividendos.
17. Registrar una reparación.
18. Consultar historial.
19. Ver auditoría.
20. Usar el sistema desde computador y teléfono.

---

# 11. Fuera del MVP

No implementar inicialmente:

- Scraping de bancos.
- Automatización de credenciales SII o TGR.
- Contabilidad tributaria completa.
- WhatsApp autónomo.
- Firma electrónica avanzada.
- Microservicios.
- Marketplace de proveedores.
- Evaluación automática de arrendatarios.
- Pagos automáticos.
- Facturación SaaS.
- Aplicación móvil nativa.

---

# 12. Roadmap funcional

## Fase 0 — Base del producto

- Organización.
- Usuarios.
- Roles.
- Propiedades.
- Auditoría.
- Configuración.

## Fase 1 — Expediente digital

- Documentos.
- Almacenamiento.
- Metadatos.
- Hash.
- Duplicados.
- Búsqueda.
- Clasificación manual.

## Fase 2 — Arriendos

- Arrendatarios.
- Contratos.
- Cobros.
- Pagos.
- Conciliación.
- Comprobantes con QR.

## Fase 3 — Control patrimonial

- Hipotecas.
- Dividendos.
- Contribuciones.
- Seguros.
- Servicios.
- Obligaciones.

## Fase 4 — Mantenimiento

- Incidentes.
- Órdenes de trabajo.
- Proveedores.
- Presupuestos.
- Garantías.
- Mejoras.

## Fase 5 — Agentes

- API de agentes.
- Agente documental.
- Agente de conciliación.
- Agente de obligaciones.
- Bandeja de aprobaciones.

## Fase 6 — SaaS

- Registro de organizaciones.
- Planes.
- Límites.
- Facturación.
- Portal de arrendatarios.
- Corredor virtual.
- Personalización.
- Soporte multiempresa.

---

# 13. Criterios de calidad

Cada funcionalidad debe incluir:

- Historias de usuario.
- Casos normales.
- Casos límite.
- Validaciones.
- Permisos.
- Auditoría.
- Estados.
- Manejo de errores.
- Pruebas.
- Migración de base de datos.
- Documentación.
- Diseño móvil.
- Criterios de aceptación verificables.

---

# 14. Formato de respuesta esperado de Codex

En cada iteración responde con:

## Diagnóstico

- Estado actual.
- Archivos relevantes.
- Riesgos.
- Decisiones necesarias.

## Plan

- Tareas pequeñas.
- Dependencias.
- Orden de ejecución.

## Implementación

- Cambios realizados.
- Archivos modificados.
- Migraciones.
- Pruebas.
- Comandos.

## Validación

- Qué se comprobó.
- Resultados.
- Limitaciones.

## Próximo paso

- Una única recomendación concreta para la siguiente iteración.

---

# 15. Primera tarea sugerida

Inspecciona el repositorio y prepara la base funcional del sistema:

1. Define el glosario del dominio.
2. Crea el modelo de organizaciones, usuarios, roles y propiedades.
3. Crea historias de usuario del MVP.
4. Define estados y transiciones.
5. Genera criterios de aceptación.
6. No implementes todavía agentes ni extracción con IA.
7. Entrega una propuesta de backlog priorizado.
