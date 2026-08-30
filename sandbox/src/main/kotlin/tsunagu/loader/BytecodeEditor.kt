package tsunagu.loader

import io.github.oshai.kotlinlogging.KotlinLogging
import org.objectweb.asm.ClassReader
import org.objectweb.asm.ClassVisitor
import org.objectweb.asm.ClassWriter
import org.objectweb.asm.FieldVisitor
import org.objectweb.asm.Handle
import org.objectweb.asm.MethodVisitor
import org.objectweb.asm.Opcodes
import org.objectweb.asm.tree.AbstractInsnNode
import org.objectweb.asm.tree.FieldInsnNode
import org.objectweb.asm.tree.InsnNode
import org.objectweb.asm.tree.MethodInsnNode
import org.objectweb.asm.tree.MethodNode
import org.objectweb.asm.tree.TypeInsnNode
import java.net.URLClassLoader
import java.nio.file.FileSystems
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardOpenOption
import kotlin.streams.asSequence

object BytecodeEditor {
    private val logger = KotlinLogging.logger {}

    fun fixAndroidClasses(jarFile: Path) {
        val classpathLoader = URLClassLoader(arrayOf(jarFile.toUri().toURL()), javaClass.classLoader)
        classpathLoader.use { loader ->
            FileSystems.newFileSystem(jarFile, null as ClassLoader?)?.use {
                val classes =
                    Files
                        .walk(it.getPath("/"))
                        .asSequence()
                        .filterNotNull()
                        .filterNot(Files::isDirectory)
                        .mapNotNull(::getClassBytes)
                        .toList()

                val instantiatedTypes = collectInstantiatedTypes(classes)
                logger.trace { "Instantiated types" to instantiatedTypes }

                classes
                    .map { entry -> transform(entry, instantiatedTypes, loader) }
                    .forEach(::write)
            }
        }
    }
    private fun collectInstantiatedTypes(classes: List<Pair<Path, ByteArray>>): Set<String> {
        val instantiated = mutableSetOf<String>()
        classes.forEach { (path, bytes) ->
            try {
                val cr = ClassReader(bytes)
                cr.accept(
                    object : ClassVisitor(Opcodes.ASM5) {
                        override fun visitMethod(
                            access: Int,
                            name: String,
                            desc: String,
                            signature: String?,
                            exceptions: Array<String?>?,
                        ): MethodVisitor =
                            object : MethodVisitor(Opcodes.ASM5) {
                                override fun visitTypeInsn(
                                    opcode: Int,
                                    type: String?,
                                ) {
                                    if (opcode == Opcodes.NEW && type != null) {
                                        instantiated.add(type)
                                    }
                                }
                            }
                    },
                    0,
                )
            } catch (e: Exception) {
                logger.error(e) { "Error scanning class for NEW instructions: $path" }
            }
        }
        return instantiated
    }

    private fun getClassBytes(path: Path): Pair<Path, ByteArray>? {
        return try {
            if (path.toString().endsWith(".class")) {
                val bytes = Files.readAllBytes(path)
                if (bytes.size < 4) {
                    return null
                }
                val cafebabe =
                    String.format(
                        "%02X%02X%02X%02X",
                        bytes[0],
                        bytes[1],
                        bytes[2],
                        bytes[3],
                    )
                if (cafebabe.lowercase() != "cafebabe") {
                    return null
                }

                path to bytes
            } else {
                null
            }
        } catch (e: Exception) {
            logger.error(e) { "Error loading class from Path: $path" }
            null
        }
    }

    private const val REPLACEMENT_PATH = "xyz/nulldev/androidcompat/replace"

    private val classesToReplace =
        listOf(
            "java/text/SimpleDateFormat",
        )

    private fun String?.replaceDirectly() =
        when (this) {
            null -> null
            in classesToReplace -> "$REPLACEMENT_PATH/$this"
            else -> this
        }

    private fun String?.replaceIndirectly(): String? {
        if (this == null) return null
        var classReference: String = this
        classesToReplace.forEach {
            classReference = classReference.replace(it, "$REPLACEMENT_PATH/$it")
        }
        return classReference
    }

    private fun computeFramesWriter(
        cr: ClassReader,
        classpathLoader: ClassLoader,
    ): ClassWriter =
        object : ClassWriter(cr, COMPUTE_FRAMES or COMPUTE_MAXS) {
            override fun getCommonSuperClass(
                type1: String,
                type2: String,
            ): String {
                if (type1 == type2) return type1
                return try {
                    val c1 = Class.forName(type1.replace('/', '.'), false, classpathLoader)
                    val c2 = Class.forName(type2.replace('/', '.'), false, classpathLoader)
                    when {
                        c1.isAssignableFrom(c2) -> type1
                        c2.isAssignableFrom(c1) -> type2
                        c1.isInterface || c2.isInterface -> "java/lang/Object"
                        else -> {
                            var x = c1
                            while (!x.isAssignableFrom(c2)) {
                                x = x.superclass ?: return "java/lang/Object"
                            }
                            x.name.replace('.', '/')
                        }
                    }
                } catch (e: Throwable) {
                    logger.trace { "Unresolvable type at merge point ($type1 / $type2), falling back to Object: ${e.message}" }
                    "java/lang/Object"
                }
            }
        }

    private fun fixBadClinitObjectInstantiation(node: MethodNode) {
        var insn: AbstractInsnNode? = node.instructions.first
        while (insn != null) {
            val newInsn = insn
            if (newInsn is TypeInsnNode && newInsn.opcode == Opcodes.NEW && newInsn.desc == "java/lang/Object") {
                val dupInsn = nextRealInsn(newInsn)
                val ctorInsn = dupInsn?.let { nextRealInsn(it) }
                val putStaticInsn = ctorInsn?.let { nextRealInsn(it) }

                if (dupInsn is InsnNode && dupInsn.opcode == Opcodes.DUP &&
                    ctorInsn is MethodInsnNode &&
                    ctorInsn.opcode == Opcodes.INVOKESPECIAL &&
                    ctorInsn.owner == "java/lang/Object" &&
                    ctorInsn.name == "<init>" &&
                    putStaticInsn is FieldInsnNode &&
                    putStaticInsn.opcode == Opcodes.PUTSTATIC
                ) {
                    val fieldType = putStaticInsn.desc.removeSurrounding("L", ";")
                    if (putStaticInsn.desc.startsWith("L") && fieldType != "java/lang/Object") {
                        logger.trace {
                            "Repointing bad clinit NEW java/lang/Object -> $fieldType" to
                                "for field ${putStaticInsn.owner}.${putStaticInsn.name}"
                        }
                        newInsn.desc = fieldType
                        ctorInsn.owner = fieldType
                    }
                }
            }
            insn = insn.next
        }
    }

    private fun nextRealInsn(insn: AbstractInsnNode): AbstractInsnNode? {
        var next = insn.next
        while (next != null && (next.opcode < 0)) {
            next = next.next
        }
        return next
    }

    private fun renamingVisitor(delegate: MethodVisitor?): MethodVisitor =
        object : MethodVisitor(Opcodes.ASM5, delegate) {
            override fun visitLdcInsn(cst: Any?) {
                logger.trace { "Ldc" to "${cst?.let { "${it::class.java.simpleName}: $it" }}" }
                super.visitLdcInsn(cst)
            }

            override fun visitTypeInsn(
                opcode: Int,
                type: String?,
            ) {
                logger.trace {
                    "Type" to "$opcode: ${type.replaceDirectly()}"
                }
                super.visitTypeInsn(
                    opcode,
                    type.replaceDirectly(),
                )
            }

            override fun visitMethodInsn(
                opcode: Int,
                owner: String?,
                name: String?,
                desc: String?,
                itf: Boolean,
            ) {
                logger.trace {
                    "Method" to "$opcode: ${owner.replaceDirectly()}: $name: ${desc.replaceIndirectly()}"
                }
                super.visitMethodInsn(
                    opcode,
                    owner.replaceDirectly(),
                    name,
                    desc.replaceIndirectly(),
                    itf,
                )
            }

            override fun visitFieldInsn(
                opcode: Int,
                owner: String?,
                name: String?,
                desc: String?,
            ) {
                logger.trace { "Field" to "$opcode: $owner: $name: ${desc.replaceIndirectly()}" }
                super.visitFieldInsn(opcode, owner, name, desc.replaceIndirectly())
            }

            override fun visitInvokeDynamicInsn(
                name: String?,
                desc: String?,
                bsm: Handle?,
                vararg bsmArgs: Any?,
            ) {
                logger.trace { "InvokeDynamic" to "$name: $desc" }
                super.visitInvokeDynamicInsn(name, desc, bsm, *bsmArgs)
            }
        }

    private fun transform(
        pair: Pair<Path, ByteArray>,
        instantiatedTypes: Set<String>,
        classpathLoader: ClassLoader,
    ): Pair<Path, ByteArray> {
        val cr = ClassReader(pair.second)
        val cw = computeFramesWriter(cr, classpathLoader)
        cr.accept(
            object : ClassVisitor(Opcodes.ASM5, cw) {
                override fun visitField(
                    access: Int,
                    name: String?,
                    desc: String?,
                    signature: String?,
                    cst: Any?,
                ): FieldVisitor? {
                    logger.trace { "CLass Field" to "${desc.replaceIndirectly()}: ${cst?.let { it::class.java.simpleName }}: $cst" }
                    return super.visitField(access, name, desc.replaceIndirectly(), signature, cst)
                }

                override fun visit(
                    version: Int,
                    access: Int,
                    name: String?,
                    signature: String?,
                    superName: String?,
                    interfaces: Array<out String>?,
                ) {
                    val isAbstract = access and Opcodes.ACC_ABSTRACT != 0
                    val isInterface = access and Opcodes.ACC_INTERFACE != 0
                    val fixedAccess =
                        if (isAbstract && !isInterface && name != null && name in instantiatedTypes) {
                            logger.trace { "Stripping ACC_ABSTRACT from instantiated class" to name }
                            access and Opcodes.ACC_ABSTRACT.inv()
                        } else {
                            access
                        }
                    logger.trace { "Visiting $name: $signature: $superName" }
                    super.visit(version, fixedAccess, name, signature, superName, interfaces)
                }

                override fun visitMethod(
                    access: Int,
                    name: String,
                    desc: String,
                    signature: String?,
                    exceptions: Array<String?>?,
                ): MethodVisitor {
                    logger.trace { "Processing method $name: ${desc.replaceIndirectly()}: $signature" }
                    val mv: MethodVisitor? =
                        super.visitMethod(
                            access,
                            name,
                            desc.replaceIndirectly(),
                            signature,
                            exceptions,
                        )

                    if (name == "<clinit>" && mv != null) {
                        val node = MethodNode(Opcodes.ASM5, access, name, desc.replaceIndirectly(), signature, exceptions)
                        val recordingVisitor = renamingVisitor(node)
                        return object : MethodVisitor(Opcodes.ASM5, recordingVisitor) {
                            override fun visitEnd() {
                                super.visitEnd()
                                fixBadClinitObjectInstantiation(node)
                                node.accept(mv)
                            }
                        }
                    }

                    return renamingVisitor(mv)
                }
            },
            0,
        )
        return pair.first to cw.toByteArray()
    }

    private fun write(pair: Pair<Path, ByteArray>) {
        Files.write(
            pair.first,
            pair.second,
            StandardOpenOption.CREATE,
            StandardOpenOption.TRUNCATE_EXISTING,
        )
    }
}
