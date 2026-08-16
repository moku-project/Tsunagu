package tsunagu.loader

import org.objectweb.asm.ClassReader
import org.objectweb.asm.ClassVisitor
import org.objectweb.asm.ClassWriter
import org.objectweb.asm.FieldVisitor
import org.objectweb.asm.MethodVisitor
import org.objectweb.asm.Opcodes
import java.nio.file.FileSystems
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardOpenOption
import kotlin.streams.asSequence

/**
 * Post-processes a dex2jar-converted extension jar.
 *
 * Based on Suwayomi's suwayomi.tachidesk.manga.impl.util.BytecodeEditor,
 * which replaces references to Android-only classes with compat
 * replacements (not currently used here — see classesToReplace below).
 *
 * Also fixes a dex2jar bug: INVOKESPECIAL calls to interface default
 * methods (e.g. kotlinx.serialization's GeneratedSerializer.
 * typeParametersSerializers()) are sometimes emitted with the ASM `itf`
 * flag set to false, producing a plain Methodref constant-pool entry
 * where the JVM requires InterfaceMethodref, causing
 * IncompatibleClassChangeError ("must be InterfaceMethodref constant").
 * Constructors (<init>) are always excluded from this fix since they are
 * never interface methods, even when the owner matches a prefix below
 * (e.g. PluginGeneratedSerialDescriptor's constructor, which is a real
 * class, not an interface).
 */
object BytecodeEditor {
    fun fixAndroidClasses(jarFile: Path) {
        FileSystems.newFileSystem(jarFile, null as ClassLoader?)?.use {
            Files
                .walk(it.getPath("/"))
                .asSequence()
                .filterNotNull()
                .filterNot(Files::isDirectory)
                .mapNotNull(::getClassBytes)
                .map(::transform)
                .forEach(::write)
        }
    }

    private fun getClassBytes(path: Path): Pair<Path, ByteArray>? {
        return try {
            if (path.toString().endsWith(".class")) {
                val bytes = Files.readAllBytes(path)
                if (bytes.size < 4) return null
                val cafebabe =
                    String.format("%02X%02X%02X%02X", bytes[0], bytes[1], bytes[2], bytes[3])
                if (cafebabe.lowercase() != "cafebabe") return null
                path to bytes
            } else {
                null
            }
        } catch (e: Exception) {
            null
        }
    }

    private const val REPLACEMENT_PATH = "tsunagu/loader/replace"

    private val interfaceInvokespecialOwnerPrefixes = listOf(
        "kotlinx/serialization/",
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

    private fun needsItfFix(
        opcode: Int,
        owner: String?,
        name: String?,
        itf: Boolean,
    ): Boolean {
        if (itf) return false
        if (opcode != Opcodes.INVOKESPECIAL) return false
        if (name == "<init>") return false // constructors are never interface methods
        return owner != null && interfaceInvokespecialOwnerPrefixes.any { owner.startsWith(it) }
    }

    private fun transform(pair: Pair<Path, ByteArray>): Pair<Path, ByteArray> {
        val cr = ClassReader(pair.second)
        val cw = ClassWriter(cr, 0)
        cr.accept(
            object : ClassVisitor(Opcodes.ASM9, cw) {
                override fun visitField(
                    access: Int,
                    name: String?,
                    desc: String?,
                    signature: String?,
                    cst: Any?,
                ): FieldVisitor? = super.visitField(access, name, desc.replaceIndirectly(), signature, cst)

                override fun visitMethod(
                    access: Int,
                    name: String,
                    desc: String,
                    signature: String?,
                    exceptions: Array<String?>?,
                ): MethodVisitor {
                    val mv: MethodVisitor? =
                        super.visitMethod(access, name, desc.replaceIndirectly(), signature, exceptions)
                    return object : MethodVisitor(Opcodes.ASM9, mv) {
                        override fun visitTypeInsn(
                            opcode: Int,
                            type: String?,
                        ) {
                            super.visitTypeInsn(opcode, type.replaceDirectly())
                        }

                        override fun visitMethodInsn(
                            opcode: Int,
                            owner: String?,
                            name: String?,
                            desc: String?,
                            itf: Boolean,
                        ) {
                            val correctedItf = itf || needsItfFix(opcode, owner, name, itf)
                            super.visitMethodInsn(
                                opcode,
                                owner.replaceDirectly(),
                                name,
                                desc.replaceIndirectly(),
                                correctedItf,
                            )
                        }

                        override fun visitFieldInsn(
                            opcode: Int,
                            owner: String?,
                            name: String?,
                            desc: String?,
                        ) {
                            super.visitFieldInsn(opcode, owner, name, desc.replaceIndirectly())
                        }
                    }
                }
            },
            0,
        )
        return pair.first to cw.toByteArray()
    }

    private fun write(pair: Pair<Path, ByteArray>) {
        Files.write(pair.first, pair.second, StandardOpenOption.CREATE, StandardOpenOption.TRUNCATE_EXISTING)
    }
}
